const handlers = new Map();

export function on(event, fn) {
  if (!handlers.has(event)) handlers.set(event, new Set());
  handlers.get(event).add(fn);
}

export function emit(event, payload) {
  const set = handlers.get(event);
  if (!set) return;
  for (const fn of set) {
    try {
      fn(payload);
    } catch (e) {
      console.error('[app] event handler failed:', event, e);
    }
  }
}