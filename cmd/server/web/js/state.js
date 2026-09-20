const FILTERS = ['all', 'active'];
const SORTS   = ['newest', 'oldest'];

function readCookie(name, allowed, fallback) {
  const m = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'));
  if (!m) return fallback;
  const v = decodeURIComponent(m[1]);
  return allowed.includes(v) ? v : fallback;
}

function writeCookie(name, value) {
  document.cookie = name + '=' + encodeURIComponent(value) +
    '; Path=/; Max-Age=' + (60 * 60 * 24 * 365) + '; SameSite=Lax';
}

export const state = {
  user:   null,
  tasks:  [],
  family: null,
  online: true,     // оптимистично; уточняется первым ответом сервера
  filter: readCookie('filter', FILTERS, 'all'),
  sort:   readCookie('sort',   SORTS,   'newest'),
};

export function setFilter(v) {
  if (!FILTERS.includes(v) || state.filter === v) return;
  state.filter = v;
  writeCookie('filter', v);
}

export function setSort(v) {
  if (!SORTS.includes(v) || state.sort === v) return;
  state.sort = v;
  writeCookie('sort', v);
}