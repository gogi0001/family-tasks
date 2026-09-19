import { els } from './dom.js';
import { state } from './state.js';
import { api } from './api.js';
import { toast } from './toast.js';
import { memberByName, memberByID } from './family.js';
import { emit } from './events.js';

const STATUSES = [
  { key: 'todo',        label: 'Надо' },
  { key: 'in_progress', label: 'В работе' },
  { key: 'done',        label: 'Готово' },
];

export async function load() {
  if (!state.family) { state.tasks = []; render(); return; }
  try {
    state.tasks = await api('/tasks');
    render();
  } catch (e) {
    if (e.status === 401 || e.status === 403) return;
    toast(e.message, 'error');
  }
}

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

  const head = document.createElement('div');
  head.className = 'task-head';

  const title = document.createElement('h2');
  title.className = 'task-title';
  title.textContent = t.title;
  head.appendChild(title);

  const del = document.createElement('button');
  del.type = 'button';
  del.className = 'icon-btn';
  del.title = 'Удалить';
  del.textContent = '×';
  del.addEventListener('click', () => removeTask(t.id));
  head.appendChild(del);
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
  els.tasks.replaceChildren();
  els.empty.hidden = state.tasks.length > 0;
  for (const t of state.tasks) els.tasks.appendChild(renderTask(t));
}

export function init() {
  els.form.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!state.user)  { emit('unauthorized'); return; }
    if (!state.family) { toast('Сначала создайте семью', 'error'); return; }

    const title       = els.title.value.trim();
    const assignee    = els.assignee.value.trim();
    const description = els.description.value.trim();

    if (!title)    { toast('Что сделать?', 'error'); els.title.focus(); return; }
    if (!assignee) { toast('Кому назначить?', 'error'); els.assignee.focus(); return; }

    try {
      await api('/tasks', {
        method: 'POST',
        body: JSON.stringify({ title, description, assignee }),
      });
      els.title.value = '';
      els.description.value = '';
      els.assignee.value = '';
      els.title.focus();
      await load();
    } catch (e) {
      if (e.status === 401) return;
      toast(e.message, 'error');
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