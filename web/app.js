const API = '/api/v1';

const STATUSES = [
  { key: 'todo',        label: 'Надо' },
  { key: 'in_progress', label: 'В работе' },
  { key: 'done',        label: 'Готово' },
];

const state = { user: null, tasks: [], family: null };

// --- Логирование ---
const log  = (...a) => console.log('[app]', ...a);
const warn = (...a) => console.warn('[app]', ...a);
const err_ = (...a) => console.error('[app]', ...a);

log('boot: script loaded');

// --- DOM ---
const $who          = document.getElementById('who-name');
const $whoLogout    = document.getElementById('who-logout');
const $modal        = document.getElementById('name-modal');
const $nameForm     = document.getElementById('name-form');
const $nameInput    = document.getElementById('name-input');

const $familyBox     = document.getElementById('family-box');
const $familyName    = document.getElementById('family-name');
const $familyCode    = document.getElementById('family-code');
const $familyMembers = document.getElementById('family-members');
const $familyGate    = document.getElementById('family-gate');
const $createForm    = document.getElementById('family-create-form');
const $createInput   = document.getElementById('family-name-input');
const $joinForm      = document.getElementById('family-join-form');
const $joinInput     = document.getElementById('family-code-input');

const $tasksArea     = document.getElementById('tasks-area');
const $form          = document.getElementById('add-form');
const $title         = document.getElementById('title');
const $assignee      = document.getElementById('assignee');
const $description   = document.getElementById('description');
const $tasks         = document.getElementById('tasks');
const $empty         = document.getElementById('empty');
const $toast         = document.getElementById('toast');

// Проверка, что все элементы найдены
(function checkDom() {
  const missing = [];
  const map = {
    'who-name': $who, 'who-logout': $whoLogout, 'name-modal': $modal,
    'name-form': $nameForm, 'name-input': $nameInput,
    'family-box': $familyBox, 'family-name': $familyName,
    'family-code': $familyCode, 'family-members': $familyMembers,
    'family-gate': $familyGate, 'family-create-form': $createForm,
    'family-name-input': $createInput, 'family-join-form': $joinForm,
    'family-code-input': $joinInput, 'tasks-area': $tasksArea,
    'add-form': $form, 'title': $title, 'assignee': $assignee,
    'description': $description, 'tasks': $tasks, 'empty': $empty,
    'toast': $toast,
  };
  for (const [id, el] of Object.entries(map)) {
    if (!el) missing.push(id);
  }
  if (missing.length) {
    err_('DOM: missing elements:', missing.join(', '));
  } else {
    log('DOM: all elements found');
  }
})();

// --- HTTP ---
async function api(path, options = {}) {
  const method = options.method || 'GET';
  const t0 = performance.now();
  log(`HTTP → ${method} ${path}`);

  let res;
  try {
    res = await fetch(API + path, {
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      ...options,
    });
  } catch (netErr) {
    err_(`HTTP ✗ ${method} ${path} — network error`, netErr);
    throw new Error('network error');
  }

  const ms = Math.round(performance.now() - t0);
  let data = null;
  if (res.status !== 204) {
    data = await res.json().catch(() => ({}));
  }

  if (!res.ok) {
    warn(`HTTP ← ${method} ${path} ${res.status} (${ms}ms)`,
      data && data.error ? data.error : '');
    const e = new Error((data && data.error) || `HTTP ${res.status}`);
    e.status = res.status;

    // Сессия протухла — сбрасываем и просим имя.
    if (res.status === 401 && path !== '/me') {
      log('HTTP: got 401 — resetting state and showing name modal');
      state.user = null;
      state.family = null;
      state.tasks = [];
      renderUser();
      renderFamily();
      render();
      showNameModal();
    }
    throw e;
  }

  log(`HTTP ← ${method} ${path} ${res.status} (${ms}ms)`, data);
  return data;
}

// --- Boot ---
async function boot() {
  log('boot: starting');
  try {
    log('boot: requesting /me');
    state.user = await api('/me');
    log('boot: identified as', state.user.name, state.user.id);
    renderUser();
    await enterFamilyFlow();
    startPolling();
    log('boot: done');
  } catch (e) {
    if (e.status === 401) {
      log('boot: not identified — showing name modal');
      showNameModal();
    } else {
      err_('boot: failed', e);
      toast(e.message, 'error');
    }
  }
}

// --- Идентификация ---
function renderUser() {
  if (state.user) {
    log('renderUser: name =', state.user.name);
    $who.textContent = 'Привет, ' + state.user.name;
    $whoLogout.hidden = false;
  } else {
    log('renderUser: no user');
    $who.textContent = '';
    $whoLogout.hidden = true;
  }
}

function showNameModal() {
  log('showNameModal');
  $modal.classList.add('open');
  setTimeout(() => $nameInput.focus(), 0);
}

function hideNameModal() {
  log('hideNameModal');
  $modal.classList.remove('open');
}

$nameForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  const name = $nameInput.value.trim();
  log('nameForm: submit, name =', JSON.stringify(name));
  if (!name) { warn('nameForm: empty name'); return; }
  try {
    state.user = await api('/me', {
      method: 'POST',
      body: JSON.stringify({ name }),
    });
    log('nameForm: success, uid =', state.user.id);
    hideNameModal();
    $nameInput.value = '';
    renderUser();
    await enterFamilyFlow();
    startPolling();
  } catch (err) {
    err_('nameForm: failed', err);
    toast(err.message, 'error');
  }
});

$whoLogout.addEventListener('click', async () => {
  log('logout: clicked');
  try {
    await api('/me', { method: 'DELETE' });
  } catch (e) {
    warn('logout: DELETE /me failed', e.message);
  }
  state.user = null;
  state.tasks = [];
  state.family = null;
  renderUser();
  renderFamily();
  render();
  showNameModal();
});

// --- Семья ---
async function enterFamilyFlow() {
  log('enterFamilyFlow: start');
  if (!state.user) {
    warn('enterFamilyFlow: no user, showing name modal');
    showNameModal();
    return;
  }
  try {
    state.family = await api('/families/me');
    log('enterFamilyFlow: family loaded =', state.family.family.name);
  } catch (e) {
    if (e.status === 404) {
      log('enterFamilyFlow: no family — showing gate');
      state.family = null;
    } else if (e.status === 401) {
      log('enterFamilyFlow: 401 — showing name modal');
      showNameModal();
      return;
    } else {
      err_('enterFamilyFlow: unexpected error', e);
      toast(e.message, 'error');
      state.family = null;
    }
  }
  renderFamily();
  await load();
}

function renderFamily() {
  const has = !!state.family;
  log('renderFamily: has family =', has);
  $familyBox.hidden = !has;
  $tasksArea.hidden = !has;
  $familyGate.hidden = has;
  if (!has) return;

  $familyName.textContent = state.family.family.name;
  $familyCode.textContent = state.family.family.inviteCode;

  $familyMembers.replaceChildren();
  for (const m of state.family.members) {
    const c = document.createElement('span');
    c.className = 'chip';
    c.textContent = m.name + (m.role === 'owner' ? ' ★' : '');
    $familyMembers.appendChild(c);
  }
}

$createForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  log('createFamily: submit');
  if (!state.user) {
    warn('createFamily: no user — showing name modal');
    showNameModal();
    return;
  }
  const name = $createInput.value.trim();
  if (!name) { warn('createFamily: empty name'); return; }
  try {
    state.family = await api('/families', {
      method: 'POST',
      body: JSON.stringify({ name }),
    });
    log('createFamily: created =', state.family.family.name,
      'code =', state.family.family.inviteCode);
    $createInput.value = '';
    renderFamily();
    await load();
  } catch (err) {
    err_('createFamily: failed', err);
    toast(err.message, 'error');
  }
});

$joinForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  log('joinFamily: submit');
  if (!state.user) {
    warn('joinFamily: no user — showing name modal');
    showNameModal();
    return;
  }
  const code = $joinInput.value.trim();
  if (!code) { warn('joinFamily: empty code'); return; }
  try {
    state.family = await api('/families/join', {
      method: 'POST',
      body: JSON.stringify({ code }),
    });
    log('joinFamily: joined =', state.family.family.name);
    $joinInput.value = '';
    renderFamily();
    await load();
  } catch (err) {
    err_('joinFamily: failed', err);
    toast(err.message, 'error');
  }
});

// --- Задачи ---
async function load() {
  if (!state.family) {
    log('load: no family — skipping');
    state.tasks = [];
    render();
    return;
  }
  try {
    const tasks = await api('/tasks');
    state.tasks = tasks;
    log('load: tasks =', tasks.length);
    render();
  } catch (e) {
    if (e.status === 401) {
      log('load: 401 — showing name modal');
      showNameModal();
      return;
    }
    if (e.status === 403) {
      log('load: 403 no family');
      return;
    }
    err_('load: failed', e);
    toast(e.message, 'error');
  }
}

function render() {
  log('render: tasks =', state.tasks.length);
  $tasks.replaceChildren();
  $empty.hidden = state.tasks.length > 0;
  for (const t of state.tasks) $tasks.appendChild(renderTask(t));
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

$form.addEventListener('submit', async (e) => {
  e.preventDefault();
  log('addTask: submit');
  if (!state.user)  { warn('addTask: no user'); showNameModal(); return; }
  if (!state.family) { warn('addTask: no family'); toast('Сначала создайте семью', 'error'); return; }

  const title       = $title.value.trim();
  const assignee    = $assignee.value.trim();
  const description = $description.value.trim();

  if (!title)    { toast('Что сделать?', 'error'); $title.focus(); return; }
  if (!assignee) { toast('Кому назначить?', 'error'); $assignee.focus(); return; }

  try {
    await api('/tasks', {
      method: 'POST',
      body: JSON.stringify({ title, description, assignee }),
    });
    log('addTask: created');
    $title.value = '';
    $description.value = '';
    $title.focus();
    await load();
  } catch (err) {
    err_('addTask: failed', err);
    if (err.status === 401) { showNameModal(); return; }
    toast(err.message, 'error');
  }
});

async function setStatus(id, status) {
  log('setStatus:', id, '→', status);
  try {
    await api(`/tasks/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    });
    await load();
  } catch (e) {
    err_('setStatus: failed', e);
    toast(e.message, 'error');
  }
}

async function removeTask(id) {
  log('removeTask:', id);
  if (!confirm('Удалить задачу?')) return;
  try {
    await api(`/tasks/${id}`, { method: 'DELETE' });
    await load();
  } catch (e) {
    err_('removeTask: failed', e);
    toast(e.message, 'error');
  }
}

// --- Toast ---
let toastTimer;
function toast(msg, kind = 'info') {
  log('toast:', kind, msg);
  $toast.textContent = msg;
  $toast.className = kind;
  $toast.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { $toast.hidden = true; }, 2500);
}

// --- Polling ---
let pollTimer = null;
function startPolling() {
  if (pollTimer) {
    log('startPolling: already running');
    return;
  }
  log('startPolling: every 5s');
  pollTimer = setInterval(() => { if (!document.hidden) load(); }, 5000);
}

// --- Погнали ---
boot();