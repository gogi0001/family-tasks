import { els } from './dom.js';
import { state } from './state.js';
import { api } from './api.js';
import { toast } from './toast.js';
import { emit } from './events.js';

const PALETTE = [
  '#e57373', '#ba68c8', '#7986cb', '#4fc3f7', '#4db6ac',
  '#81c784', '#ffd54f', '#ffb74d', '#a1887f', '#90a4ae',
];

export function renderUser() {
  if (state.user) {
    const color = state.user.color || '#90a4ae';
    const letter = (state.user.name || '?').trim().charAt(0).toUpperCase();
    els.avatar.style.background = color;
    els.avatarLetter.textContent = letter;
    els.avatar.hidden = false;
    els.who.textContent = state.user.name;
    els.whoLogout.hidden = false;
  } else {
    els.avatar.hidden = true;
    els.who.textContent = '';
    els.whoLogout.hidden = true;
    els.colorPicker.hidden = true;
  }
}

function renderColorPicker() {
  els.colorGrid.replaceChildren();
  for (const c of PALETTE) {
    const b = document.createElement('button');
    b.type = 'button';
    b.className = 'color-swatch' + (state.user?.color === c ? ' active' : '');
    b.style.background = c;
    b.addEventListener('click', () => setColor(c));
    els.colorGrid.appendChild(b);
  }
}

async function setColor(color) {
  try {
    state.user = await api('/me', {
      method: 'PATCH',
      body: JSON.stringify({ color }),
    });
    els.colorPicker.hidden = true;
    emit('user-updated');
  } catch (e) {
    toast(e.message, 'error');
  }
}

export function init() {
  els.avatar.addEventListener('click', (e) => {
    e.stopPropagation();
    if (!state.user) return;
    renderColorPicker();
    els.colorPicker.hidden = !els.colorPicker.hidden;
  });

  document.addEventListener('click', (e) => {
    if (els.colorPicker.hidden) return;
    if (els.colorPicker.contains(e.target) || e.target === els.avatar) return;
    els.colorPicker.hidden = true;
  });
}