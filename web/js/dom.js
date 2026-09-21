const $ = id => document.getElementById(id);

export const log = (...a) => console.log('[app]', ...a);
export const warn = (...a) => console.warn('[app]', ...a);
export const err = (...a) => console.error('[app]', ...a);

export const els = {
  // Шапка
  connBadge: $('conn-badge'),
  connText: $('conn-text'),
  who: $('who-name'),
  whoLogout: $('who-logout'),
  avatar: $('who-avatar'),
  avatarLetter: $('who-avatar-letter'),
  colorPicker: $('color-picker'),
  colorGrid: $('color-grid'),

  // Идентификация
  modal: $('name-modal'),
  nameForm: $('name-form'),
  nameInput: $('name-input'),

  // Семья
  familyBox: $('family-box'),
  familyName: $('family-name'),
  familyCode: $('family-code'),
  familyRegen: $('family-regen'),
  familyMembers: $('family-members'),
  familyGate: $('family-gate'),
  createForm: $('family-create-form'),
  createInput: $('family-name-input'),
  joinForm: $('family-join-form'),
  joinInput: $('family-code-input'),

  // Задачи и тулбар
  tasksArea: $('tasks-area'),
  tasks: $('tasks'),
  empty: $('empty'),

  taskFilter: $('task-filter'),
  taskSort: $('task-sort'),

  filterToggle: $('filter-toggle'),
  filterPanel: $('filter-panel'),
  filterStatus: $('filter-status'),
  filterCreator: $('filter-creator'),
  filterAssignee: $('filter-assignee'),
  filterFrom: $('filter-created-from'),
  filterTo: $('filter-created-to'),
  filterReset: $('filter-reset'),

  // Модалка добавления
  addTaskFab: $('add-task-fab'),
  addTaskModal: $('add-task-modal'),
  addTaskCancel: $('add-task-cancel'),
  form: $('add-form'),
  title: $('title'),
  assignee: $('assignee'),
  due: $('due'),
  description: $('description'),

  toast: $('toast'),
};

export function checkDom() {
  const missing = Object.entries(els).filter(([, v]) => !v).map(([k]) => k);
  if (missing.length) err('DOM: missing:', missing.join(', '));
  else log('DOM: all elements found');
}