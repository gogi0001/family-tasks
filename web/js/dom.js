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

  // Auth
  authModal: $('auth-modal'),
  authTabLogin: $('auth-tab-login'),
  authTabRegister: $('auth-tab-register'),
  authLoginForm: $('auth-login-form'),
  authRegForm: $('auth-register-form'),
  loginEmail: $('login-email'),
  loginPassword: $('login-password'),
  registerEmail: $('register-email'),
  registerName: $('register-name'),
  registerPassword: $('register-password'),
  registerInvite: $('register-invite'),
  inviteHint: $('invite-hint'),

  // Семья
  familyOpen: $('family-open'),
  familyGate: $('family-gate'),
  createForm: $('family-create-form'),
  createInput: $('family-name-input'),
  joinForm: $('family-join-form'),
  joinInput: $('family-code-input'),

  familyModal: $('family-modal'),
  familyModalTitle: $('family-modal-title'),
  familyModalCode: $('family-modal-code'),
  familyModalRegen: $('family-modal-regen'),
  familyModalMembers: $('family-modal-members'),
  familyModalClose: $('family-modal-close'),
  familyModalCloseBtn: $('family-modal-close-btn'),

  invitesSection: $('invites-section'),
  invitesList: $('invites-list'),
  inviteCreate: $('invite-create'),

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

  // Модалка задачи
  addTaskFab: $('add-task-fab'),
  addTaskModal: $('add-task-modal'),
  addTaskCancel: $('add-task-cancel'),
  form: $('add-form'),
  title: $('title'),
  assignee: $('assignee'),
  due: $('due'),
  dueField: $('due-field'),
  description: $('description'),

  // Повторяющиеся
  taskRecurring: $('task-recurring'),
  recurringFields: $('recurring-fields'),
  recurringType: $('recurring-type'),
  recurringTime: $('recurring-time'),
  recurringInterval: $('recurring-interval'),
  recurringWeekly: $('recurring-weekly'),
  recurringDaily: $('recurring-daily'),
  recurringMonthly: $('recurring-monthly'),
  recurringDay: $('recurring-day'),
  weekdays: document.querySelectorAll('.weekdays input[data-wd]'),

  templatesOpen: $('templates-open'),
  templatesModal: $('templates-modal'),
  templatesList: $('templates-list'),
  templatesEmpty: $('templates-empty'),
  templatesClose: $('templates-close'),
  templatesCloseBtn: $('templates-close-btn'),

  // Вложения
  uploadInput: $('upload-input'),
  lightbox: $('lightbox'),
  lightboxImg: $('lightbox-img'),

  toast: $('toast'),
};

export function checkDom() {
  const missing = Object.entries(els)
    .filter(([, v]) => v == null)
    .map(([k]) => k);
  if (missing.length) err('DOM: missing:', missing.join(', '));
  else log('DOM: all elements found');
}