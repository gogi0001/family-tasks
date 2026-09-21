const SORT_OPTIONS = ['newest', 'oldest'];
const QUICK_OPTIONS = ['all', 'my-active', 'overdue'];
const STATUS_OPTIONS = ['all', 'active', 'todo', 'in_progress', 'done'];

const FILTER_KEY = 'family-tasks.filter.v1';
const SORT_KEY = 'family-tasks.sort.v1';

function defaultFilter() {
  return {
    quick: 'all',
    status: 'all',
    creator: '',
    assignee: '',
    createdFrom: '',
    createdTo: '',
    open: false,
  };
}

function loadFilter() {
  try {
    const raw = localStorage.getItem(FILTER_KEY);
    if (!raw) return defaultFilter();
    const parsed = JSON.parse(raw);
    const f = { ...defaultFilter(), ...parsed };
    if (!QUICK_OPTIONS.includes(f.quick)) f.quick = 'all';
    if (!STATUS_OPTIONS.includes(f.status)) f.status = 'all';
    return f;
  } catch {
    return defaultFilter();
  }
}

function saveFilter() {
  try {
    localStorage.setItem(FILTER_KEY, JSON.stringify(state.filter));
  } catch { }
}

function loadSort() {
  const v = localStorage.getItem(SORT_KEY);
  return SORT_OPTIONS.includes(v) ? v : 'newest';
}

function saveSort() {
  try { localStorage.setItem(SORT_KEY, state.sort); } catch { }
}

export const state = {
  user: null,
  tasks: [],
  family: null,
  online: true,
  filter: loadFilter(),
  sort: loadSort(),
};

export function setFilter(patch) {
  Object.assign(state.filter, patch);
  saveFilter();
}

export function resetAdvancedFilter() {
  setFilter({
    status: 'all',
    creator: '',
    assignee: '',
    createdFrom: '',
    createdTo: '',
  });
}

export function setSort(v) {
  if (!SORT_OPTIONS.includes(v) || state.sort === v) return;
  state.sort = v;
  saveSort();
}

export function isAdvancedDefault() {
  const f = state.filter;
  return (
    f.status === 'all' &&
    !f.creator &&
    !f.assignee &&
    !f.createdFrom &&
    !f.createdTo
  );
}