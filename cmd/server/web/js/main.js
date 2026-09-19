import { state } from './state.js';
import { checkDom, log } from './dom.js';
import { api } from './api.js';
import { on } from './events.js';
import * as user   from './user.js';
import * as family from './family.js';
import * as tasks  from './tasks.js';

checkDom();

// --- Реакции на события ---

// import * as tasks from './tasks.js';

on('unauthorized', () => {
  log('event: unauthorized — reset');
  tasks.closeAddModal();          // ← добавили
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
  tasks.closeAddModal();          // ← добавили
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

// --- Init ---
user.init();
family.init();
tasks.init();

// --- Boot ---
async function boot() {
  try {
    state.user = await api('/me');
    user.renderUser();
    await family.enterFamilyFlow();
    startPolling();
  } catch (e) {
    if (e.status === 401) user.showNameModal();
    else console.error('[app] boot failed', e);
  }
}

let pollTimer = null;
function startPolling() {
  if (pollTimer) return;
  pollTimer = setInterval(() => { if (!document.hidden) tasks.load(); }, 5000);
}

boot();