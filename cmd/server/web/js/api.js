import { log, warn, err } from './dom.js';
import { emit } from './events.js';

const API = '/api/v1';

export async function api(path, options = {}) {
  const method = options.method || 'GET';
  const t0 = performance.now();
  log(`HTTP → ${method} ${path}`);

  let res;
  try {
    res = await fetch(API + path, {
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      ...options,
    });
  } catch (netErr) {
    err(`HTTP ✗ ${method} ${path}`, netErr);
    throw new Error('network error');
  }

  const ms = Math.round(performance.now() - t0);
  let data = null;
  if (res.status !== 204) data = await res.json().catch(() => ({}));

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