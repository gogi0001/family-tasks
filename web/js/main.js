import { state } from './state.js';
import { checkDom, log } from './dom.js';
import { api } from './api.js';
import { on } from './events.js';
import { toast } from './toast.js';
import * as user from './user.js';
import * as family from './family.js';
import * as tasks from './tasks.js';
import * as connection from './connection.js';
import * as sse from './sse.js';
import * as attachments from './attachments.js';

checkDom();

// --- Реакции на события ---

on('unauthorized', () => {
  log('event: unauthorized — reset');
  tasks.closeAddModal();
  family.closeFamilyModal();
  sse.stop();
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
  family.closeFamilyModal();
  sse.stop();
  state.family = null;
  state.tasks = [];
  user.renderUser();
  family.renderFamily();
  tasks.render();
  user.showNameModal();
});

on('identified', async () => {
  log('event: identified');
  await family.enterFamilyFlow();
  sse.start();
  startRefreshFallback();
});

on('family-updated', async () => {
  log('event: family-updated');
  family.renderFamily();
  tasks.refreshFilterOptions();
  await tasks.load();
});

on('user-updated', () => {
  log('event: user-updated');
  user.renderUser();
  family.renderFamily();
  tasks.render();
});

on('connection-changed', async (online) => {
  log('event: connection-changed →', online ? 'online' : 'offline');
  state.online = online;
  connection.renderConnection();

  if (!online) return;

  toast('Связь восстановлена', 'info');
  try {
    state.user = await api('/me');
    user.renderUser();
    await family.enterFamilyFlow();
    sse.start();
  } catch (e) {
    if (e.status === 401) user.showNameModal();
  }
});

// --- SSE ---

on('sse-event', () => {
  tasks.load();
});

on('sse-open', () => {
  // Догоняем всё, что изменилось, пока не было соединения.
  tasks.load();
});

on('attachments-changed', () => {
  log('event: attachments-changed');
  tasks.load();
});

// --- Init ---

user.init();
family.init();
tasks.init();
attachments.init();      // ← новое
connection.renderConnection();

// --- Boot ---

async function boot() {
  try {
    state.user = await api('/me');
    user.renderUser();
    await family.enterFamilyFlow();
    sse.start();
    startRefreshFallback();
  } catch (e) {
    if (e.status === 401) user.showNameModal();
    else if (e.status === 0) { /* offline */ }
    else console.error('[app] boot failed', e);
  }
}

// --- Refresh fallback ---
// SSE даёт мгновенные обновления. Раз в 60 секунд обновляемся на всякий
// случай, плюс при возврате во вкладку/окно. Если SSE не работает —
// данные всё равно будут свежими.

let refreshTimer = null;
let lastRefreshAt = 0;

function refreshIfStale(minIntervalMs = 1000) {
  const now = Date.now();
  if (now - lastRefreshAt < minIntervalMs) return;
  lastRefreshAt = now;
  tasks.load();
}

function startRefreshFallback() {
  if (refreshTimer) return;
  log('refresh fallback: 60s + visibility/focus');

  refreshTimer = setInterval(() => {
    if (!document.hidden) refreshIfStale(30000);
  }, 60000);

  document.addEventListener('visibilitychange', () => {
    if (!document.hidden) refreshIfStale();
  });

  window.addEventListener('focus', () => {
    refreshIfStale();
  });
}

boot();