import { els } from './dom.js';
import { state } from './state.js';
import { api } from './api.js';
import { toast } from './toast.js';
import { emit } from './events.js';

export async function enterFamilyFlow() {
  if (!state.user) { emit('unauthorized'); return; }
  try {
    state.family = await api('/families/me');
  } catch (e) {
    if (e.status === 404) {
      state.family = null;
    } else if (e.status === 401) {
      return; // api.js уже отправил 'unauthorized'
    } else {
      toast(e.message, 'error');
      state.family = null;
    }
  }
  emit('family-updated');
}

export function renderFamily() {
  const has = !!state.family;
  els.familyBox.hidden = !has;
  els.tasksArea.hidden = !has;
  els.familyGate.hidden = has;

  if (!has) {
    els.familyMembers.replaceChildren();
    els.assignee.replaceChildren();
    return;
  }

  els.familyName.textContent = state.family.family.name;
  els.familyCode.textContent = state.family.family.inviteCode;

  els.familyMembers.replaceChildren();
  for (const m of state.family.members) {
    const c = document.createElement('span');
    c.className = 'chip';
    const dot = document.createElement('span');
    dot.className = 'dot';
    dot.style.background = m.color || '#90a4ae';
    c.appendChild(dot);
    c.appendChild(document.createTextNode(m.name + (m.role === 'owner' ? ' ★' : '')));
    els.familyMembers.appendChild(c);
  }

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

export function init() {
  els.createForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!state.user) { emit('unauthorized'); return; }
    const name = els.createInput.value.trim();
    if (!name) return;
    try {
      state.family = await api('/families', {
        method: 'POST',
        body: JSON.stringify({ name }),
      });
      els.createInput.value = '';
      emit('family-updated');
    } catch (e) {
      toast(e.message, 'error');
    }
  });

  els.joinForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!state.user) { emit('unauthorized'); return; }
    const code = els.joinInput.value.trim();
    if (!code) return;
    try {
      state.family = await api('/families/join', {
        method: 'POST',
        body: JSON.stringify({ code }),
      });
      els.joinInput.value = '';
      emit('family-updated');
    } catch (e) {
      toast(e.message, 'error');
    }
  });
}