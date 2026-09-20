import { els } from './dom.js';
import { state } from './state.js';

export function renderConnection() {
  const online = state.online;
  els.connBadge.classList.toggle('conn-online', online);
  els.connBadge.classList.toggle('conn-offline', !online);
  els.connText.textContent = online ? 'онлайн' : 'нет связи';
  els.connBadge.title = online
    ? 'Связь с сервером активна'
    : 'Нет связи с сервером. Пробуем переподключиться…';
}