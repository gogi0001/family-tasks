import { log, warn, err } from './dom.js';
import { emit } from './events.js';

const API = '/api/v1';
const TIMEOUT_MS = 8000;

let online = true;

function setOnline(v) {
  if (online === v) return;
  online = v;
  emit('connection-changed', v);
}

export function isOnline() { return online; }

export async function api(path, options = {}) {
  const method = options.method || 'GET';
  const isForm = options.body instanceof FormData;
  const t0 = performance.now();
  log(`HTTP → ${method} ${path}`);

  const headers = isForm
    ? { ...(options.headers || {}) }
    : { 'Content-Type': 'application/json', ...(options.headers || {}) };

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), TIMEOUT_MS);

  let res;
  try {
    res = await fetch(API + path, {
      credentials: 'same-origin',
      ...options,
      headers,
      signal: controller.signal,
    });
  } catch (netErr) {
    err(`HTTP ✗ ${method} ${path} — network error`, netErr);
    setOnline(false);
    const e = new Error('Сервер недоступен');
    e.status = 0;
    e.offline = true;
    throw e;
  } finally {
    clearTimeout(timeoutId);
  }

  // Любой ответ — значит связь есть.
  // 
  setOnline(true);

  const ms = Math.round(performance.now() - t0);
  let data = null;
  if (res.status !== 204) {
    data = await res.json().catch(() => ({}));
  }

  if (!res.ok) {
    warn(`HTTP ← ${method} ${path} ${res.status} (${ms}ms)`, data?.error || '');
    const e = new Error(data?.error || `HTTP ${res.status}`);
    e.status = res.status;
    if (res.status === 401 && path !== '/me') emit('unauthorized');
    throw e;
  }
  log(`HTTP ← ${method} ${path} ${res.status} (${ms}ms)`, data);
  return data;
}