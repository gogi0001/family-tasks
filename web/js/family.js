import { els } from './dom.js';
import { state } from './state.js';
import { api } from './api.js';
import { toast } from './toast.js';
import { emit } from './events.js';
import * as invites from './invites.js';

export async function enterFamilyFlow() {
  if (!state.user) { emit('unauthenticated'); return; }
  try {
    state.family = await api('/families/me');
  } catch (e) {
    if (e.status === 404) {
      state.family = null;
    } else if (e.status === 401) {
      return;
    } else if (e.status === 0) {
      return;
    } else {
      toast(e.message, 'error');
      state.family = null;
    }
  }
  emit('family-updated');
}

export function myRole() {
  const me = state.family?.members.find(m => m.id === state.user?.id);
  return me?.role || '';
}

export function isOwner() {
  return myRole() === 'owner';
}

// --- Модалка ---

function openFamilyModal() {
  if (!state.family) return;
  els.familyModal.classList.add('open');
  renderFamilyModal();
  invites.loadInvites();
}

export function closeFamilyModal() {
  els.familyModal.classList.remove('open');
}

function isFamilyModalOpen() {
  return els.familyModal.classList.contains('open');
}

function renderFamilyModal() {
  const view = state.family;
  if (!view) return;

  els.familyModalTitle.textContent = view.family.name;
  els.familyModalCode.textContent = view.family.inviteCode;
  els.familyModalRegen.hidden = !isOwner();

  els.familyModalMembers.replaceChildren();
  for (const m of view.members) {
    const c = document.createElement('span');
    c.className = 'chip';

    const dot = document.createElement('span');
    dot.className = 'dot';
    dot.style.background = m.color || '#90a4ae';
    c.appendChild(dot);

    c.appendChild(document.createTextNode(m.name + (m.role === 'owner' ? ' ★' : '')));

    const canRemove = isOwner() && m.id !== state.user?.id;
    if (canRemove) {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'chip-remove';
      btn.title = 'Удалить участника';
      btn.textContent = '×';
      btn.addEventListener('click', () => removeMember(m.id, m.name));
      c.appendChild(btn);
    }

    els.familyModalMembers.appendChild(c);
  }
}

// --- Рендер главного экрана ---

export function renderFamily() {
  const has = !!state.family;

  els.familyGate.hidden = has;
  els.tasksArea.hidden = !has;
  els.addTaskFab.hidden = !has;
  els.familyOpen.hidden = !has;

  if (!has) {
    els.assignee.replaceChildren();
    closeFamilyModal();
    return;
  }

  if (isFamilyModalOpen()) renderFamilyModal();

  const previous = els.assignee.value;
  els.assignee.replaceChildren();
  const ph = document.createElement('option');
  ph.value = '';
  ph.textContent = 'Кому?';
  ph.disabled = true;
  ph.selected = true;
  els.assignee.appendChild(ph);
  for (const m of state.family.members) {
    const opt = document.createElement('option');
    opt.value = m.name;
    opt.textContent = m.name;
    if (m.name === previous) opt.selected = true;
    els.assignee.appendChild(opt);
  }
}

export function memberByName(name) {
  return state.family?.members.find(m => m.name === name);
}

export function memberByID(id) {
  return state.family?.members.find(m => m.id === id);
}

// --- Действия ---

async function removeMember(id, name) {
  if (!confirm(`Удалить «${name}» из семьи?`)) return;
  try {
    await api(`/families/members/${id}`, { method: 'DELETE' });
    await enterFamilyFlow();
  } catch (e) {
    if (e.status === 0) return;
    toast(e.message, 'error');
  }
}

async function regenerateCode() {
  if (!confirm('Старый код перестанет работать. Перевыпустить?')) return;
  try {
    await api('/families/invite/regenerate', { method: 'POST' });
    await enterFamilyFlow();
  } catch (e) {
    if (e.status === 0) return;
    toast(e.message, 'error');
  }
}

// --- Init ---

export function init() {
  els.familyOpen.addEventListener('click', openFamilyModal);
  els.familyModalClose.addEventListener('click', closeFamilyModal);
  els.familyModalCloseBtn.addEventListener('click', closeFamilyModal);
  els.familyModal.addEventListener('click', (e) => {
    if (e.target === els.familyModal) closeFamilyModal();
  });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && isFamilyModalOpen()) closeFamilyModal();
  });

  els.familyModalRegen.addEventListener('click', regenerateCode);

  els.createForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!state.user) { emit('unauthenticated'); return; }
    const name = els.createInput.value.trim();
    if (!name) return;
    try {
      state.family = await api('/families', {
        method: 'POST',
        body: JSON.stringify({ name }),
      });
      els.createInput.value = '';
      emit('family-updated');
    } catch (err) {
      if (err.status === 0) return;
      toast(err.message, 'error');
    }
  });

  els.joinForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!state.user) { emit('unauthenticated'); return; }
    const invite = els.joinInput.value.trim();
    if (!invite) return;
    try {
      state.family = await api('/families/join', {
        method: 'POST',
        body: JSON.stringify({ invite }),
      });
      els.joinInput.value = '';
      emit('family-updated');
    } catch (err) {
      if (err.status === 0) return;
      toast(err.message, 'error');
    }
  });
}