import { els } from './dom.js';

let timer;
export function toast(msg, kind = 'info') {
  els.toast.textContent = msg;
  els.toast.className = kind;
  els.toast.hidden = false;
  clearTimeout(timer);
  timer = setTimeout(() => { els.toast.hidden = true; }, 2500);
}