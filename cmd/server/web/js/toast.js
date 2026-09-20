import { els } from './dom.js';
import { state } from './state.js';

let timer;
let lastMsg = '';
let lastAt = 0;
const DEDUP_MS = 5000;

export function toast(msg, kind = 'info') {
  // Если связи нет — красный бейдж в шапке уже всё сказал.
  if (kind === 'error' && !state.online) return;

  // Одно и то же сообщение подряд — не чаще раза в 5 секунд.
  const now = Date.now();
  if (msg === lastMsg && now - lastAt < DEDUP_MS) return;
  lastMsg = msg;
  lastAt = now;

  els.toast.textContent = msg;
  els.toast.className = kind;
  els.toast.hidden = false;
  clearTimeout(timer);
  timer = setTimeout(() => { els.toast.hidden = true; }, 2500);
}