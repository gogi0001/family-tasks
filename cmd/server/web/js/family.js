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
      return;
    } else if (e.status === 0) {
      return; // offline — ждём восстановления
    } else {
      toast(e.message, 'error');
      state.family = null;
    }
  }
  emit('family-updated');
}

// --- Роль текущего пользователя (берём из состава семьи) ---

export function myRole() {
  const me = state.family?.members.find(m => m.id === state.user?.id);
  return me?.role || '';
}

export function isOwner() {
  return myRole() === 'owner';
}

// --- Рендер ---

export function renderFamily() {
  const has = !!state.family;

  els.familyBox.hidden  = !has;
  els.tasksArea.hidden  = !has;
  els.familyGate.hidden = has;
  els.addTaskFab.hidden = !has;

  if (!has) {
    els.familyMembers.replaceChildren();
    els.assignee.replaceChildren();
    els.familyRegen.hidden = true;
    return;
  }

  els.familyName.textContent = state.family.family.name;
  els.familyCode.textContent = state.family.family.inviteCode;
  els.familyRegen.hidden = !isOwner();

  els.familyMembers.replaceChildren();
  for (const m of state.family.members) {
    const c = document.createElement('span');
    c.className = 'chip';

    const dot = document.createElement('span');
    dot.className = 'dot';
    dot.style.background = m.color || '#90a4ae';
    c.appendChild(dot);

    c.appendChild(document.createTextNode(m.name + (m.role === 'owner' ? ' ★' : '')));

    // Кнопка «выгнать» — только owner и не для себя
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

    els.familyMembers.appendChild(c);
  }

  // Выпадающий список исполнителей
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
    await enterFamilyFlow(); // перечитать /families/me
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
  els.familyRegen.addEventListener('click', regenerateCode);

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
      if (e.status === 0) return;
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
      if (e.status === 0) return;
      toast(e.message, 'error');
    }
  });
}