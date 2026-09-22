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

// --- Модалка семьи ---

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

    // Кнопка сброса пароля — только owner и не для себя
    const canReset = isOwner() && m.id !== state.user?.id;
    if (canReset) {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'chip-remove';
      btn.title = 'Сбросить пароль участнику';
      btn.textContent = '🔑';
      btn.addEventListener('click', () => resetMemberPassword(m.id, m.name));
      c.appendChild(btn);
    }

    // Кнопка удаления из семьи — только owner и не для себя
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

async function resetMemberPassword(id, name) {
  if (!confirm(
    `Сбросить пароль участнику «${name}»?\n\n` +
    `Все его активные сессии будут завершены, ` +
    `а старый пароль перестанет работать.`
  )) return;

  try {
    const res = await api(`/families/members/${id}/reset-password`, {
      method: 'POST',
    });
    openNewPasswordModal(res.name, res.email, res.password);
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

// --- Модалка нового пароля ---

function openNewPasswordModal(name, email, password) {
  els.newpassTitle.textContent = 'Новый пароль';
  els.newpassSubtitle.textContent = `Для ${name} (${email})`;
  els.newpassValue.textContent = password;
  els.newpassModal.classList.add('open');
}

export function closeNewPassModal() {
  els.newpassModal.classList.remove('open');
  els.newpassValue.textContent = '';
}

function isNewPassOpen() {
  return els.newpassModal.classList.contains('open');
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

  // Модалка нового пароля
  els.newpassClose.addEventListener('click', closeNewPassModal);
  els.newpassModal.addEventListener('click', (e) => {
    if (e.target === els.newpassModal) closeNewPassModal();
  });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && isNewPassOpen()) closeNewPassModal();
  });
  els.newpassCopy.addEventListener('click', async () => {
    const pwd = els.newpassValue.textContent;
    if (!pwd) return;
    try {
      await navigator.clipboard.writeText(pwd);
      toast('Пароль скопирован', 'info');
    } catch {
      window.prompt('Скопируйте пароль:', pwd);
    }
  });

  // Создание семьи
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

  // Присоединение по инвайту
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