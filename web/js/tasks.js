import { els } from './dom.js';
import { state, setFilter, setSort, resetAdvancedFilter, isAdvancedDefault } from './state.js';
import { api } from './api.js';
import { toast } from './toast.js';
import { memberByName, memberByID, isOwner } from './family.js';
import { emit } from './events.js';

const STATUSES = [
  { key: 'todo',        label: 'Надо' },
  { key: 'in_progress', label: 'В работе' },
  { key: 'done',        label: 'Готово' },
];

// --- Модалка добавления ---

export function openAddModal() {
  els.addTaskModal.classList.add('open');
  setTimeout(() => els.title.focus(), 0);
}

export function closeAddModal() {
  els.addTaskModal.classList.remove('open');
  els.title.value = '';
  els.description.value = '';
  els.due.value = '';
}

function isAddModalOpen() {
  return els.addTaskModal.classList.contains('open');
}

// --- Утилиты форматирования ---

function fmtDateTime(iso) {
  if (!iso) return '';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '';
  return d.toLocaleString('ru-RU', {
    day: '2-digit', month: '2-digit',
    hour: '2-digit', minute: '2-digit',
  });
}

function toRFC3339(localValue) {
  if (!localValue) return '';
  const d = new Date(localValue);
  if (isNaN(d.getTime())) return '';
  return d.toISOString();
}

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

// --- Счётчики на быстрых кнопках ---

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

  if (t.dueAt) {
    const due = document.createElement('span');
    due.textContent = 'Срок ' + fmtDateTime(t.dueAt);
    if (isOverdue(t)) due.classList.add('overdue');
    else if (t.status === 'done') due.classList.add('done-muted');
    dates.appendChild(due);
  } else {
    const noDue = document.createElement('span');
    noDue.textContent = 'Бессрочно';
    dates.appendChild(noDue);
  }
  card.appendChild(dates);

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
  els.filterStatus.value   = state.filter.status;
  els.filterCreator.value  = state.filter.creator;
  els.filterAssignee.value = state.filter.assignee;
  els.filterFrom.value     = state.filter.createdFrom;
  els.filterTo.value       = state.filter.createdTo;
}

export function refreshFilterOptions() {
  const members = state.family?.members || [];

  const creator  = fillFilterSelect(els.filterCreator,  members, state.filter.creator);
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

// --- Init ---

export function init() {
  els.addTaskFab.addEventListener('click', openAddModal);
  els.addTaskCancel.addEventListener('click', closeAddModal);

  els.addTaskModal.addEventListener('click', (e) => {
    if (e.target === els.addTaskModal) closeAddModal();
  });

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && isAddModalOpen()) closeAddModal();
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
    if (!state.user)   { emit('unauthorized'); return; }
    if (!state.family) { toast('Сначала создайте семью', 'error'); return; }

    const title       = els.title.value.trim();
    const assignee    = els.assignee.value.trim();
    const description = els.description.value.trim();
    const dueAt       = toRFC3339(els.due.value);

    if (!title)    { toast('Что сделать?', 'error'); els.title.focus(); return; }
    if (!assignee) { toast('Кому назначить?', 'error'); els.assignee.focus(); return; }

    try {
      await api('/tasks', {
        method: 'POST',
        body: JSON.stringify({ title, description, assignee, dueAt }),
      });
      closeAddModal();
      await load();
    } catch (err) {
      if (err.status === 401) return;
      toast(err.message, 'error');
    }
  });

  updateQuickButtons();
  syncFilterPanelFromState();
  applyFilterPanelVisibility();
  updateFilterToggle();

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