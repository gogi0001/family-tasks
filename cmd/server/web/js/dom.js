const $ = id => document.getElementById(id);

export const log  = (...a) => console.log('[app]', ...a);
export const warn = (...a) => console.warn('[app]', ...a);
export const err  = (...a) => console.error('[app]', ...a);

export const els = {
  who:          $('who-name'),
  whoLogout:    $('who-logout'),
  avatar:       $('who-avatar'),
  avatarLetter: $('who-avatar-letter'),
  colorPicker:  $('color-picker'),
  colorGrid:    $('color-grid'),

  modal:        $('name-modal'),
  nameForm:     $('name-form'),
  nameInput:    $('name-input'),

  familyBox:     $('family-box'),
  familyName:    $('family-name'),
  familyCode:    $('family-code'),
  familyMembers: $('family-members'),
  familyGate:    $('family-gate'),
  createForm:    $('family-create-form'),
  createInput:   $('family-name-input'),
  joinForm:      $('family-join-form'),
  joinInput:     $('family-code-input'),

  tasksArea:    $('tasks-area'),
  form:         $('add-form'),
  title:        $('title'),
  assignee:     $('assignee'),
  description:  $('description'),
  tasks:        $('tasks'),
  empty:        $('empty'),
  toast:        $('toast'),
};

export function checkDom() {
  const missing = Object.entries(els).filter(([, v]) => !v).map(([k]) => k);
  if (missing.length) err('DOM: missing:', missing.join(', '));
  else log('DOM: all elements found');
}