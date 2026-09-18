const API = '/api/v1';

const STATUSES = [
  { key: 'todo',        label: 'Надо' },
  { key: 'in_progress', label: 'В работе' },
  { key: 'done',        label: 'Готово' },
];

const $me          = document.getElementById('me');
const $form        = document.getElementById('add-form');
const $title       = document.getElementById('title');
const $assignee    = document.getElementById('assignee');
const $description = document.getElementById('description');
const $tasks       = document.getElementById('tasks');
const $empty       = document.getElementById('empty');
const $toast       = document.getElementById('toast');

let tasks = [];

// --- «Я» сохраняем в localStorage (задел под идентификацию) ---
$me.value = localStorage.getItem('me') || '';
$me.addEventListener('change', () => {
  localStorage.setItem('me', $me.value.trim());
});

// --- HTTP-обёртка ---
async function api(path, options = {}) {
  const res = await fetch(API + path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

// --- Загрузка и рендер ---
async function load() {
  try {
    tasks = await api('/tasks');
    render();
  } catch (e) {
    toast(e.message, 'error');
  }
}

function render() {
  $tasks.replaceChildren();
  $empty.hidden = tasks.length > 0;
  for (const t of tasks) $tasks.appendChild(renderTask(t));
}

function renderTask(t) {
  const card = document.createElement('article');
  card.className = 'task';
  card.dataset.status = t.status;

  // Заголовок + кнопка удаления
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

  // Описание
  if (t.description) {
    const desc = document.createElement('p');
    desc.className = 'task-desc';
    desc.textContent = t.description;
    card.appendChild(desc);
  }

  // Мета: кому и от кого
  const meta = document.createElement('div');
  meta.className = 'task-meta';

  const assignee = document.createElement('span');
  assignee.className = 'chip';
  assignee.textContent = '→ ' + t.assignee;
  meta.appendChild(assignee);

  if (t.createdBy) {
    const author = document.createElement('span');
    author.className = 'chip muted';
    author.textContent = 'от ' + t.createdBy;
    meta.appendChild(author);
  }

  card.appendChild(meta);

  // Кнопки статусов
  const statuses = document.createElement('div');
  statuses.className = 'statuses';
  for (const s of STATUSES) {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'status-btn' + (t.status === s.key ? ' active' : '');
    btn.dataset.status = s.key;
    btn.textContent = s.label;
    btn.addEventListener('click', () => setStatus(t.id, s.key));
    statuses.appendChild(btn);
  }
  card.appendChild(statuses);

  return card;
}

// --- Действия ---
$form.addEventListener('submit', async (e) => {
  e.preventDefault();

  const title       = $title.value.trim();
  const assignee    = $assignee.value.trim();
  const description = $description.value.trim();
  const createdBy   = $me.value.trim();

  if (!createdBy) { toast('Укажите своё имя в шапке', 'error'); $me.focus(); return; }
  if (!title)     { toast('Что сделать?', 'error'); $title.focus(); return; }
  if (!assignee)  { toast('Кому назначить?', 'error'); $assignee.focus(); return; }

  try {
    await api('/tasks', {
      method: 'POST',
      body: JSON.stringify({ title, description, assignee, createdBy }),
    });
    $title.value = '';
    $description.value = '';
    $title.focus();
    await load();
  } catch (err) {
    toast(err.message, 'error');
  }
});

async function setStatus(id, status) {
  try {
    await api(`/tasks/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    });
    await load();
  } catch (e) {
    toast(e.message, 'error');
  }
}

async function removeTask(id) {
  if (!confirm('Удалить задачу?')) return;
  try {
    await api(`/tasks/${id}`, { method: 'DELETE' });
    await load();
  } catch (e) {
    toast(e.message, 'error');
  }
}

// --- Тост ---
let toastTimer;
function toast(msg, kind = 'info') {
  $toast.textContent = msg;
  $toast.className = kind;
  $toast.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { $toast.hidden = true; }, 2500);
}

// --- Первичная загрузка и мягкий поллинг ---
load();
setInterval(() => { if (!document.hidden) load(); }, 5000);