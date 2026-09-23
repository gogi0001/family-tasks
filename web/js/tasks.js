import { els } from './dom.js';
import { state, setFilter, setSort, resetAdvancedFilter, isAdvancedDefault } from './state.js';
import { api } from './api.js';
import { toast } from './toast.js';
import { memberByName, memberByID, isOwner } from './family.js';
import { emit } from './events.js';
import * as attachments from './attachments.js';
import { fmtDateTime } from './format.js';

const STATUSES = [
  { key: 'todo', label: 'Надо' },
  { key: 'in_progress', label: 'В работе' },
  { key: 'done', label: 'Готово' },
];

let dueEditTaskId = null;

// --- Модалка добавления ---

export function openAddModal() {
  els.addTaskModal.classList.add('open');
  setTimeout(() => els.title.focus(), 0);
}

export function closeAddModal() {
  els.addTaskModal.classList.remove('open');
  els.title.value = '';
  els.description.value = '';
  els.dueDate.value = '';
  els.dueTime.value = '';
  els.taskRecurring.checked = false;
  updateRecurringUI();
}

function isAddModalOpen() {
  return els.addTaskModal.classList.contains('open');
}

// --- Модалка срока ---

function openDueModal(task) {
  dueEditTaskId = task.id;
  els.dueModalTitle.textContent = 'Изменить срок';
  els.dueModalTask.textContent = task.title;

  if (task.dueAt) {
    const d = new Date(task.dueAt);
    const pad = n => String(n).padStart(2, '0');
    els.dueInputDate.value = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
    els.dueInputTime.value = `${pad(d.getHours())}:${pad(d.getMinutes())}`;
  } else {
    els.dueInputDate.value = '';
    els.dueInputTime.value = '';
  }

  els.dueModal.classList.add('open');
  setTimeout(() => els.dueInputDate.focus(), 0);
}

export function closeDueModal() {
  els.dueModal.classList.remove('open');
  dueEditTaskId = null;
  els.dueInputDate.value = '';
  els.dueInputTime.value = '';
}

function isDueModalOpen() {
  return els.dueModal.classList.contains('open');
}

// --- Сборка dueAt из двух полей ---
//   оба пусты        → '' (бессрочно)
//   только дата      → дата + текущее время
//   только время     → сегодняшняя дата + время
//   оба заполнены    → как есть
function buildDueAt(dateEl, timeEl) {
  const d = dateEl.value;   // YYYY-MM-DD
  const t = timeEl.value;   // HH:MM

  if (!d && !t) return '';

  const now = new Date();
  let year = now.getFullYear();
  let month = now.getMonth();
  let day = now.getDate();
  let hours = now.getHours();
  let minutes = now.getMinutes();

  if (d) {
    const [y, m, dd] = d.split('-').map(Number);
    year = y; month = m - 1; day = dd;
  }
  if (t) {
    const [hh, mm] = t.split(':').map(Number);
    hours = hh; minutes = mm;
  }

  const dt = new Date(year, month, day, hours, minutes, 0, 0);
  if (isNaN(dt.getTime())) return '';
  return dt.toISOString();
}

// --- Повторение в модалке добавления ---

function updateRecurringUI() {
  const on = els.taskRecurring.checked;

  els.recurringFields.hidden = !on;
  els.dueField.hidden = on;

  const t = els.recurringType.value;
  els.recurringDaily.hidden = t !== 'daily';
  els.recurringWeekly.hidden = t !== 'weekly';
  els.recurringMonthly.hidden = t !== 'monthly';
}

function buildRule() {
  const type = els.recurringType.value;
  const time = els.recurringTime.value || '09:00';

  switch (type) {
    case 'daily': {
      const interval = Math.max(1, Math.min(365, parseInt(els.recurringInterval.value, 10) || 1));
      return { type: 'daily', interval, time };
    }
    case 'weekly': {
      const days = [];
      for (const cb of els.weekdays) {
        if (cb.checked) days.push(parseInt(cb.dataset.wd, 10));
      }
      return { type: 'weekly', weekdays: days, time };
    }
    case 'monthly': {
      const day = Math.max(1, Math.min(31, parseInt(els.recurringDay.value, 10) || 1));
      return { type: 'monthly', dayOfMonth: day, time };
    }
  }
  return null;
}

function validateRule(rule) {
  if (!rule) return 'Не удалось определить правило';
  if (rule.type === 'weekly' && (!rule.weekdays || rule.weekdays.length === 0)) {
    return 'Выберите хотя бы один день недели';
  }
  return null;
}

// --- Утилиты ---

function isOverdue(task) {
  if (!task.dueAt) return false;
  if (task.status === 'done') return false;
  return new Date(task.dueAt).getTime() < Date.now();
}

// --- Фильтрация ---

function applyQuick(task, quick) {
  const me = state.user?.name;
  switch (quick) {
    case 'my-active':
      if (!me) return true;
      return (task.createdBy === me || task.assignee === me) && task.status !== 'done';
    case 'overdue':
      return isOverdue(task);
    case 'all':
    default:
      return true;
  }
}

function applyAdvanced(task, f) {
  if (f.status === 'active') {
    if (task.status === 'done') return false;
  } else if (f.status !== 'all') {
    if (task.status !== f.status) return false;
  }
  if (f.creator && task.createdBy !== f.creator) return false;
  if (f.assignee && task.assignee !== f.assignee) return false;
  if (f.createdFrom) {
    const from = new Date(f.createdFrom + 'T00:00:00').getTime();
    if (new Date(task.createdAt).getTime() < from) return false;
  }
  if (f.createdTo) {
    const to = new Date(f.createdTo + 'T23:59:59.999').getTime();
    if (new Date(task.createdAt).getTime() > to) return false;
  }
  return true;
}

function visibleTasks() {
  const f = state.filter;
  const list = state.tasks.filter(t => applyQuick(t, f.quick) && applyAdvanced(t, f));
  list.sort((a, b) => {
    const aT = new Date(a.createdAt).getTime();
    const bT = new Date(b.createdAt).getTime();
    return state.sort === 'oldest' ? aT - bT : bT - aT;
  });
  return list;
}

// --- Счётчики ---

function countForQuick() {
  const me = state.user?.name;
  let all = 0, my = 0, overdue = 0;
  for (const t of state.tasks) {
    all++;
    if (me && (t.createdBy === me || t.assignee === me) && t.status !== 'done') my++;
    if (isOverdue(t)) overdue++;
  }
  return { all, 'my-active': my, overdue };
}

function renderQuickCounts() {
  const counts = countForQuick();
  for (const btn of els.taskFilter.querySelectorAll('button[data-quick]')) {
    const n = counts[btn.dataset.quick] ?? 0;
    let badge = btn.querySelector('.badge');
    if (n > 0) {
      if (!badge) {
        badge = document.createElement('span');
        badge.className = 'badge';
        btn.appendChild(badge);
      }
      badge.textContent = String(n);
    } else if (badge) {
      badge.remove();
    }
  }
}

// --- Загрузка ---

export async function load() {
  if (!state.family) { state.tasks = []; render(); return; }
  try {
    state.tasks = await api('/tasks');
    render();
  } catch (e) {
    if (e.status === 401 || e.status === 403 || e.status === 0) return;
    toast(e.message, 'error');
  }
}

// --- Рендер ---

function coloredChip(prefix, name, color, muted = false) {
  const chip = document.createElement('span');
  chip.className = 'chip' + (muted ? ' muted' : '');
  if (color) {
    const dot = document.createElement('span');
    dot.className = 'dot';
    dot.style.background = color;
    chip.appendChild(dot);
  }
  chip.appendChild(document.createTextNode(prefix + name));
  return chip;
}

function renderAttachments(taskId, list) {
  const wrap = document.createElement('div');
  wrap.className = 'task-attachments';

  for (const a of list) {
    const cell = document.createElement('div');
    cell.className = 'task-attachment';

    const img = document.createElement('img');
    img.src = a.url;
    img.alt = a.filename || '';
    img.loading = 'lazy';
    img.addEventListener('click', () => attachments.openLightbox(a));
    cell.appendChild(img);

    const rm = document.createElement('button');
    rm.type = 'button';
    rm.className = 'attachment-remove';
    rm.title = 'Удалить';
    rm.textContent = '×';
    rm.addEventListener('click', async (e) => {
      e.stopPropagation();
      if (!confirm('Удалить вложение?')) return;
      try {
        await api(`/tasks/${taskId}/attachments/${a.id}`, { method: 'DELETE' });
        emit('attachments-changed');
      } catch (err) {
        if (err.status === 0) return;
        toast(err.message, 'error');
      }
    });
    cell.appendChild(rm);

    wrap.appendChild(cell);
  }

  const addBtn = document.createElement('button');
  addBtn.type = 'button';
  addBtn.className = 'task-attachment attachment-add'
    + (list.length === 0 ? ' is-empty' : '');
  addBtn.title = 'Прикрепить фото';
  addBtn.textContent = '+';
  addBtn.addEventListener('click', () => attachments.pickFilesFor(taskId));
  wrap.appendChild(addBtn);

  return wrap;
}

function renderTask(t) {
  const card = document.createElement('article');
  card.className = 'task';
  card.dataset.status = t.status;
  card.dataset.id = t.id;

  const head = document.createElement('div');
  head.className = 'task-head';

  const title = document.createElement('h2');
  title.className = 'task-title';
  title.textContent = t.title;
  head.appendChild(title);

  if (isOwner() || t.createdBy === state.user?.name) {
    const del = document.createElement('button');
    del.type = 'button';
    del.className = 'icon-btn';
    del.title = 'Удалить';
    del.textContent = '×';
    del.addEventListener('click', () => removeTask(t.id));
    head.appendChild(del);
  }
  card.appendChild(head);

  if (t.description) {
    const desc = document.createElement('p');
    desc.className = 'task-desc';
    desc.textContent = t.description;
    card.appendChild(desc);
  }

  const meta = document.createElement('div');
  meta.className = 'task-meta';

  const assignee = memberByName(t.assignee);
  meta.appendChild(coloredChip('→ ', t.assignee, assignee?.color));

  if (t.createdBy) {
    const author = memberByName(t.createdBy);
    meta.appendChild(coloredChip('от ', t.createdBy, author?.color, true));
  }
  card.appendChild(meta);

  const dates = document.createElement('div');
  dates.className = 'task-dates';

  if (t.templateId) {
    const recur = document.createElement('span');
    recur.className = 'chip small';
    recur.title = 'Из повторяющейся задачи';
    recur.textContent = '⟳';
    dates.appendChild(recur);
  }

  const created = document.createElement('span');
  created.textContent = 'Начата ' + fmtDateTime(t.createdAt);
  dates.appendChild(created);

  if (t.statusUpdatedAt) {
    const updater = t.statusUpdatedBy ? memberByID(t.statusUpdatedBy) : null;
    const s = document.createElement('span');
    s.textContent = 'Статус ' + fmtDateTime(t.statusUpdatedAt) +
      (updater ? ' (' + updater.name + ')' : '');
    dates.appendChild(s);
  }

  const canEditDue = isOwner() || t.createdBy === state.user?.name;

  const dueWrap = document.createElement('span');
  if (t.dueAt) {
    dueWrap.textContent = 'Срок ' + fmtDateTime(t.dueAt);
    if (isOverdue(t)) dueWrap.classList.add('overdue');
    else if (t.status === 'done') dueWrap.classList.add('done-muted');
  } else {
    dueWrap.textContent = 'Бессрочно';
  }
  dates.appendChild(dueWrap);

  if (canEditDue) {
    const edit = document.createElement('button');
    edit.type = 'button';
    edit.className = 'due-edit';
    edit.title = 'Изменить срок';
    edit.textContent = '✎';
    edit.addEventListener('click', () => openDueModal(t));
    dates.appendChild(edit);
  }
  card.appendChild(dates);

  card.appendChild(renderAttachments(t.id, t.attachments || []));

  const updater = t.statusUpdatedBy ? memberByID(t.statusUpdatedBy) : null;
  const activeColor = updater?.color || null;

  const statuses = document.createElement('div');
  statuses.className = 'statuses';
  for (const s of STATUSES) {
    const btn = document.createElement('button');
    btn.type = 'button';
    const active = t.status === s.key;
    btn.className = 'status-btn' + (active ? ' active' : '');
    btn.dataset.status = s.key;
    btn.textContent = s.label;
    if (active && activeColor) {
      btn.style.background = activeColor;
      btn.style.borderColor = activeColor;
      btn.style.color = '#fff';
    }
    btn.addEventListener('click', () => setStatus(t.id, s.key));
    statuses.appendChild(btn);
  }
  card.appendChild(statuses);

  return card;
}

export function render() {
  const list = visibleTasks();
  els.tasks.replaceChildren();
  els.empty.hidden = list.length > 0;

  if (state.tasks.length === 0) {
    els.empty.textContent = 'Пока задач нет.';
  } else if (list.length === 0) {
    els.empty.textContent = 'По фильтру ничего не найдено.';
  }

  for (const t of list) els.tasks.appendChild(renderTask(t));

  renderQuickCounts();
  updateFilterToggle();
}

// --- Тулбар ---

function updateQuickButtons() {
  for (const b of els.taskFilter.querySelectorAll('button[data-quick]')) {
    b.classList.toggle('active', b.dataset.quick === state.filter.quick);
  }
}

function updateFilterToggle() {
  const advanced = !isAdvancedDefault();
  els.filterToggle.classList.toggle('active', advanced);
  els.filterToggle.classList.toggle('open', state.filter.open);
  els.filterToggle.title = advanced
    ? 'Подробный фильтр (включены ограничения)'
    : 'Подробный фильтр';
}

function applyFilterPanelVisibility() {
  els.filterPanel.hidden = !state.filter.open;
}

function syncFilterPanelFromState() {
  els.filterStatus.value = state.filter.status;
  els.filterCreator.value = state.filter.creator;
  els.filterAssignee.value = state.filter.assignee;
  els.filterFrom.value = state.filter.createdFrom;
  els.filterTo.value = state.filter.createdTo;
}

export function refreshFilterOptions() {
  const members = state.family?.members || [];

  const creator = fillFilterSelect(els.filterCreator, members, state.filter.creator);
  const assignee = fillFilterSelect(els.filterAssignee, members, state.filter.assignee);

  if (creator !== state.filter.creator || assignee !== state.filter.assignee) {
    setFilter({ creator, assignee });
    render();
  }
}

function fillFilterSelect(select, members, currentValue) {
  select.replaceChildren();
  const any = document.createElement('option');
  any.value = '';
  any.textContent = 'Любой';
  select.appendChild(any);

  for (const m of members) {
    const opt = document.createElement('option');
    opt.value = m.name;
    opt.textContent = m.name;
    select.appendChild(opt);
  }
  const valid = currentValue === '' || members.some(m => m.name === currentValue);
  select.value = valid ? currentValue : '';
  return valid ? currentValue : '';
}

// --- Подсветка задачи по deep link ---

export function highlightTask(id) {
  if (!id) return;
  console.log('[app] highlightTask:', id);

  const attempt = (triesLeft) => {
    document.querySelectorAll('.task.task-highlight')
      .forEach(el => el.classList.remove('task-highlight'));

    const el = document.querySelector(`.task[data-id="${CSS.escape(id)}"]`);
    if (!el) {
      if (triesLeft > 0) {
        setTimeout(() => attempt(triesLeft - 1), 500);
      } else {
        console.warn('[app] highlightTask: not found', id);
      }
      return;
    }
    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    el.classList.add('task-highlight');
    setTimeout(() => el.classList.remove('task-highlight'), 4000);
    history.replaceState(null, '', location.pathname + location.search);
  };
  attempt(30);
}

// --- Init ---

export function init() {
  els.addTaskFab.addEventListener('click', openAddModal);
  els.addTaskCancel.addEventListener('click', closeAddModal);

  els.addTaskModal.addEventListener('click', (e) => {
    if (e.target === els.addTaskModal) closeAddModal();
  });

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && isAddModalOpen()) closeAddModal();
    if (e.key === 'Escape' && isDueModalOpen()) closeDueModal();
  });

  els.taskRecurring.addEventListener('change', updateRecurringUI);
  els.recurringType.addEventListener('change', updateRecurringUI);

  // Модалка срока
  els.dueCancel.addEventListener('click', closeDueModal);
  els.dueModal.addEventListener('click', (e) => {
    if (e.target === els.dueModal) closeDueModal();
  });
  els.dueForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!dueEditTaskId) return;
    const dueAt = buildDueAt(els.dueInputDate, els.dueInputTime);
    await applyDueChange(dueEditTaskId, dueAt);
  });
  els.dueClear.addEventListener('click', async () => {
    if (!dueEditTaskId) return;
    await applyDueChange(dueEditTaskId, '');
  });

  els.taskFilter.addEventListener('click', (e) => {
    const btn = e.target.closest('button[data-quick]');
    if (!btn) return;
    setFilter({ quick: btn.dataset.quick });
    updateQuickButtons();
    render();
  });

  els.taskSort.addEventListener('change', () => {
    setSort(els.taskSort.value);
    render();
  });

  els.filterToggle.addEventListener('click', () => {
    setFilter({ open: !state.filter.open });
    applyFilterPanelVisibility();
    updateFilterToggle();
  });

  els.filterStatus.addEventListener('change', () => {
    setFilter({ status: els.filterStatus.value });
    render();
  });
  els.filterCreator.addEventListener('change', () => {
    setFilter({ creator: els.filterCreator.value });
    render();
  });
  els.filterAssignee.addEventListener('change', () => {
    setFilter({ assignee: els.filterAssignee.value });
    render();
  });
  els.filterFrom.addEventListener('change', () => {
    setFilter({ createdFrom: els.filterFrom.value });
    render();
  });
  els.filterTo.addEventListener('change', () => {
    setFilter({ createdTo: els.filterTo.value });
    render();
  });
  els.filterReset.addEventListener('click', () => {
    resetAdvancedFilter();
    syncFilterPanelFromState();
    render();
  });

  els.form.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!state.user) { emit('unauthenticated'); return; }
    if (!state.family) { toast('Сначала создайте семью', 'error'); return; }

    const title = els.title.value.trim();
    const assignee = els.assignee.value.trim();
    const description = els.description.value.trim();
    const recurring = els.taskRecurring.checked;

    if (!title) { toast('Что сделать?', 'error'); els.title.focus(); return; }
    if (!assignee) { toast('Кому назначить?', 'error'); els.assignee.focus(); return; }

    if (recurring) {
      const rule = buildRule();
      const err = validateRule(rule);
      if (err) { toast(err, 'error'); return; }
      try {
        const created = await api('/templates', {
          method: 'POST',
          body: JSON.stringify({ title, description, assignee, rule }),
        });
        closeAddModal();
        const next = fmtDateTime(created.nextRunAt);
        toast(next ? `Правило создано. Первая задача: ${next}` : 'Правило создано', 'info');
        emit('template-changed');
      } catch (err) {
        if (err.status === 401) { emit('unauthenticated'); return; }
        toast(err.message, 'error');
      }
      return;
    }

    const dueAt = buildDueAt(els.dueDate, els.dueTime);
    try {
      await api('/tasks', {
        method: 'POST',
        body: JSON.stringify({ title, description, assignee, dueAt }),
      });
      closeAddModal();
      await load();
    } catch (err) {
      if (err.status === 401) { emit('unauthenticated'); return; }
      toast(err.message, 'error');
    }
  });

  updateQuickButtons();
  syncFilterPanelFromState();
  applyFilterPanelVisibility();
  updateFilterToggle();
  updateRecurringUI();
  els.taskSort.value = state.sort;
}

async function setStatus(id, status) {
  try {
    await api(`/tasks/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    });
    await load();
  } catch (e) { toast(e.message, 'error'); }
}

async function removeTask(id) {
  if (!confirm('Удалить задачу?')) return;
  try {
    await api(`/tasks/${id}`, { method: 'DELETE' });
    await load();
  } catch (e) { toast(e.message, 'error'); }
}

async function applyDueChange(taskId, dueAt) {
  try {
    await api(`/tasks/${taskId}`, {
      method: 'PATCH',
      body: JSON.stringify({ dueAt }),
    });
    closeDueModal();
    await load();
  } catch (e) {
    if (e.status === 0) return;
    toast(e.message, 'error');
  }
}