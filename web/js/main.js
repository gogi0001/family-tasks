import { state } from './state.js';
import { checkDom, log } from './dom.js';
import { api } from './api.js';
import { on } from './events.js';
import { toast } from './toast.js';
import * as user from './user.js';
import * as family from './family.js';
import * as tasks from './tasks.js';
import * as connection from './connection.js';

checkDom();

// --- Реакции на события ---

on('unauthorized', () => {
  log('event: unauthorized — reset');
  tasks.closeAddModal();
  state.user = null;
  state.family = null;
  state.tasks = [];
  user.renderUser();
  family.renderFamily();
  tasks.render();
  user.showNameModal();
});

on('logged-out', () => {
  log('event: logged-out');
  tasks.closeAddModal();
  state.family = null;
  state.tasks = [];
  user.renderUser();
  family.renderFamily();
  tasks.render();
  user.showNameModal();
});

on('identified', async () => {
  log('event: identified — loading family');
  await family.enterFamilyFlow();
  startPolling();
});

on('family-updated', async () => {
  log('event: family-updated');
  family.renderFamily();
  await tasks.load();
});

on('user-updated', () => {
  log('event: user-updated — rerender');
  user.renderUser();
  family.renderFamily();
  tasks.render();
});

on('connection-changed', async (online) => {
  log('event: connection-changed →', online ? 'online' : 'offline');
  state.online = online;
  connection.renderConnection();

  if (!online) return;

  // Связь вернулась — догоняем состояние.
  toast('Связь восстановлена', 'info');
  try {
    state.user = await api('/me');
    user.renderUser();
    await family.enterFamilyFlow();
  } catch (e) {
    if (e.status === 401) user.showNameModal();
    // network error — снова уйдём в offline, следующий reconnect попробует опять
  }
});

// --- Init ---
user.init();
family.init();
tasks.init();
connection.renderConnection();

// --- Boot ---
async function boot() {
  try {
    state.user = await api('/me');
    user.renderUser();
    await family.enterFamilyFlow();
    startPolling();
    handleTaskHash();

  } catch (e) {
    if (e.status === 401) user.showNameModal();
    else if (e.status === 0) { /* offline — ждём восстановления */ }
    else console.error('[app] boot failed', e);
  }
}

// --- Refresh ---
let pollTimer = null;
let lastRefreshAt = 0;

function refreshIfStale(minIntervalMs = 1000) {
  const now = Date.now();
  if (now - lastRefreshAt < minIntervalMs) return;
  lastRefreshAt = now;
  tasks.load();
}

function startPolling() {
  if (pollTimer) return;
  log('startPolling: every 5s + on visibility/focus');

  pollTimer = setInterval(() => {
    if (!document.hidden) refreshIfStale(3000);
  }, 5000);

  document.addEventListener('visibilitychange', () => {
    if (!document.hidden) refreshIfStale();
  });

  window.addEventListener('focus', () => {
    refreshIfStale();
  });
}
// --- Deep link: #task=<id> ---

function handleTaskHash() {
  const m = location.hash.match(/(?:^#|&)task=([^&]+)/);
  if (!m) return;
  const id = decodeURIComponent(m[1]);
  log('deep link: task =', id);
  tasks.highlightTask(id);
}

window.addEventListener('hashchange', handleTaskHash);

boot();