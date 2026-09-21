import { state } from './state.js';
import { checkDom, log } from './dom.js';
import { api } from './api.js';
import { on } from './events.js';
import { toast } from './toast.js';
import * as user from './user.js';
import * as auth from './auth.js';
import * as family from './family.js';
import * as tasks from './tasks.js';
import * as invites from './invites.js';
import * as connection from './connection.js';
import * as attachments from './attachments.js';
import * as sse from './sse.js';

checkDom();

// --- События ---

on('unauthenticated', () => {
  log('event: unauthenticated — reset');
  tasks.closeAddModal();
  tasks.closeDueModal();          // ← новое
  family.closeFamilyModal();
  attachments.closeLightbox();
  sse.stop();
  state.user = null;
  state.family = null;
  state.tasks = [];
  user.renderUser();
  family.renderFamily();
  tasks.render();
  auth.openAuthModal();
});

on('logged-out', () => {
  log('event: logged-out');
  tasks.closeAddModal();
  tasks.closeDueModal();          // ← новое
  family.closeFamilyModal();
  attachments.closeLightbox();
  sse.stop();
  state.family = null;
  state.tasks = [];
  user.renderUser();
  family.renderFamily();
  tasks.render();
  auth.openAuthModal();
});

on('identified', async () => {
  log('event: identified');
  user.renderUser();
  await family.enterFamilyFlow();
  sse.start();
  startRefreshFallback();
});

on('family-updated', async () => {
  log('event: family-updated');
  family.renderFamily();
  tasks.refreshFilterOptions();
  await tasks.load();
  handleTaskHash();
});

on('user-updated', () => {
  log('event: user-updated');
  user.renderUser();
  family.renderFamily();
  tasks.render();
});

on('attachments-changed', () => {
  log('event: attachments-changed');
  tasks.load();
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
    if (e.status === 401) auth.openAuthModal('login');
  }
});

on('sse-event', () => {
  tasks.load();
});

on('sse-open', () => {
  tasks.load();
});

// --- Init ---

user.init();
auth.init();
family.init();
tasks.init();
invites.init();
attachments.init();
connection.renderConnection();

// --- Deep link: #task=<id> ---

function handleTaskHash() {
  const m = location.hash.match(/(?:^#|&)task=([^&]+)/);
  if (!m) return;
  const id = decodeURIComponent(m[1]);
  log('deep link: task =', id);
  tasks.highlightTask(id);
}

window.addEventListener('hashchange', handleTaskHash);

// --- Boot ---

async function boot() {
  try {
    state.user = await api('/me');
    user.renderUser();
    await family.enterFamilyFlow();
    sse.start();
    startRefreshFallback();
    handleTaskHash();
  } catch (e) {
    if (e.status === 401) {
      auth.openAuthModal();
    } else if (e.status === 0) {
      // offline
    } else {
      console.error('[app] boot failed', e);
    }
  }
}

// --- Refresh fallback ---

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