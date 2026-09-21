import { els } from './dom.js';
import { state, setFilter, setSort } from './state.js';
import { api } from './api.js';
import { toast } from './toast.js';
import { memberByName, memberByID, isOwner } from './family.js';
import { emit } from './events.js';

const STATUSES = [
  { key: 'todo', label: 'Надо' },
  { key: 'in_progress', label: 'В работе' },
  { key: 'done', label: 'Готово' },
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

// --- Список: фильтр и сортировка ---

function visibleTasks() {
  let ts = state.tasks;
  if (state.filter === 'active') {
    ts = ts.filter(t => t.status !== 'done');
  }
  const sorted = [...ts].sort((a, b) => {
    const aT = new Date(a.createdAt).getTime();
    const bT = new Date(b.createdAt).getTime();
    return state.sort === 'oldest' ? aT - bT : bT - aT;
  });
  return sorted;
}

// --- Загрузка ---

export async function load() {
  if (!state.family) { state.tasks = []; render(); return; }
  try {
    state.tasks = await api('/tasks');
    render();
  } catch (e) {
    if (e.status === 401 || e.status === 403) return;
    if (e.status === 0) return;          // offline — индикатор уже показал
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
  card.dataset.id = t.id;              // ← новое

  const head = document.createElement('div');
  head.className = 'task-head';

  const title = document.createElement('h2');
  title.className = 'task-title';
  title.textContent = t.title;
  head.appendChild(title);


  const canDelete = isOwner() || t.createdBy === state.user?.name;
  if (canDelete) {
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

  // Мета: кому, от кого
  const meta = document.createElement('div');
  meta.className = 'task-meta';

  const assignee = memberByName(t.assignee);
  meta.appendChild(coloredChip('→ ', t.assignee, assignee?.color));

  if (t.createdBy) {
    const author = memberByName(t.createdBy);
    meta.appendChild(coloredChip('от ', t.createdBy, author?.color, true));
  }
  card.appendChild(meta);

  // Даты: создана, статус, срок
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

  // Статусы
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
  els.empty.textContent = state.tasks.length === 0
    ? 'Пока задач нет.'
    : 'По фильтру ничего не найдено.';
  for (const t of list) els.tasks.appendChild(renderTask(t));
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

  // --- Синхронизация UI с состоянием (cookie могла задать не дефолт) ---
  for (const b of els.taskFilter.querySelectorAll('button[data-filter]')) {
    b.classList.toggle('active', b.dataset.filter === state.filter);
  }
  els.taskSort.value = state.sort;

  // --- Фильтр ---
  els.taskFilter.addEventListener('click', (e) => {
    const btn = e.target.closest('button[data-filter]');
    if (!btn) return;
    setFilter(btn.dataset.filter);
    for (const b of els.taskFilter.querySelectorAll('button')) {
      b.classList.toggle('active', b === btn);
    }
    render();
  });

  // --- Сортировка ---
  els.taskSort.addEventListener('change', () => {
    setSort(els.taskSort.value);
    render();
  });

  // --- Submit формы добавления ---
  els.form.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!state.user) { emit('unauthorized'); return; }
    if (!state.family) { toast('Сначала создайте семью', 'error'); return; }

    const title = els.title.value.trim();
    const assignee = els.assignee.value.trim();
    const description = els.description.value.trim();
    const dueAt = toRFC3339(els.due.value);

    if (!title) { toast('Что сделать?', 'error'); els.title.focus(); return; }
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

// highlightTask — скроллит к карточке и подсвечивает её.
// Если карточки ещё нет (задачи не загружены), повторяет попытку до 5 секунд.
export function highlightTask(id) {
  if (!id) return;

  const attempt = (triesLeft) => {
    document.querySelectorAll('.task.task-highlight')
      .forEach(el => el.classList.remove('task-highlight'));

    const el = document.querySelector(`.task[data-id="${CSS.escape(id)}"]`);
    if (!el) {
      if (triesLeft > 0) setTimeout(() => attempt(triesLeft - 1), 500);
      else console.log('[app] highlightTask: not found', id);
      return;
    }

    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    el.classList.add('task-highlight');
    setTimeout(() => el.classList.remove('task-highlight'), 4000);

    // Убираем hash, чтобы F5 не подсвечивал снова
    history.replaceState(null, '', location.pathname + location.search);
  };

  attempt(10);
}