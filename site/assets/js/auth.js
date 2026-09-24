// ============================================================================
//  Управление ролями
// ============================================================================
//  Три роли:
//    viewer  — пользователь, только смотрит
//    worker  — работник, может перемещать лиды между стадиями (drag-n-drop)
//    manager — менеджер, всё то же + редактирование полей в карточке лида
// ============================================================================

const STORAGE_KEY = 'currentRole';
const DEFAULT_ROLE = 'viewer';

export const ROLES = {
  viewer:  { id: 'viewer',  title: 'Пользователь' },
  worker:  { id: 'worker',  title: 'Работник' },
  manager: { id: 'manager', title: 'Менеджер' },
};

// Матрица прав:  role → set of actions
const PERMISSIONS = {
  viewer:  new Set(['view']),
  worker:  new Set(['view', 'move']),
  manager: new Set(['view', 'move', 'edit', 'create', 'import']),
  // импорт CSV логично доступен менеджеру; если правила другие — поправим.
};

let _currentRole = DEFAULT_ROLE;
const _listeners = new Set();

export function initAuth() {
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved && ROLES[saved]) _currentRole = saved;
}

export function getRole() {
  return _currentRole;
}

export function setRole(role) {
  if (!ROLES[role]) throw new Error(`Unknown role: ${role}`);
  _currentRole = role;
  localStorage.setItem(STORAGE_KEY, role);
  _listeners.forEach((fn) => fn(role));
}

export function onRoleChange(fn) {
  _listeners.add(fn);
  return () => _listeners.delete(fn);
}

/** Проверить можно ли текущей роли выполнить действие */
export function can(action) {
  return PERMISSIONS[_currentRole]?.has(action) ?? false;
}

if (typeof window !== 'undefined') {
  window.__auth = { getRole, setRole, can, ROLES };
}
