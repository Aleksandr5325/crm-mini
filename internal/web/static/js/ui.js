// Общие помощники интерфейса: экранирование, форматы, стадии, справочники, уведомления, диалоги.

import { api, ApiError } from './api.js';

// esc экранирует текст перед вставкой в HTML — защита от XSS:
// данные лидов приходят от пользователей и из файлов импорта.
export function esc(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

// ---------- Стадии ----------

export const STAGES = [
  { code: 'new', label: 'Новый' },
  { code: 'qualified', label: 'Квалифицирован' },
  { code: 'proposal', label: 'Предложение' },
  { code: 'won', label: 'Выигран' },
  { code: 'lost', label: 'Проигран' },
];

const stageLabels = Object.fromEntries(STAGES.map((s) => [s.code, s.label]));

export function stageLabel(code) {
  return stageLabels[code] ?? code;
}

export function stageBadge(code) {
  return `<span class="badge st-${esc(code)}">${esc(stageLabel(code))}</span>`;
}

// ---------- Форматы ----------

const dateTime = new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium', timeStyle: 'short' });
const dateOnly = new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium' });
const decimal = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 1 });

export const fmt = {
  dateTime: (iso) => (iso ? dateTime.format(new Date(iso)) : '—'),
  date: (iso) => (iso ? dateOnly.format(new Date(iso)) : '—'),
  number: (n) => (n === null || n === undefined ? '—' : decimal.format(n)),
  days: (from, to) => decimal.format((new Date(to) - new Date(from)) / 86_400_000),
};

// ---------- Справочники: источники и сотрудники ----------

let refsPromise = null;

// refs загружает источники и сотрудников один раз и отдаёт удобные словари.
export function refs() {
  if (!refsPromise) {
    refsPromise = Promise.all([api.sources(), api.employees()]).then(([sources, employees]) => ({
      sources,
      employees,
      sourceName: Object.fromEntries(sources.map((s) => [s.code, s.name])),
      ownerName: Object.fromEntries(employees.map((e) => [e.id, e.full_name])),
    }));
    refsPromise.catch(() => { refsPromise = null; });
  }
  return refsPromise;
}

// resetRefs заставляет загрузить справочники заново — например, после добавления источника.
export function resetRefs() {
  refsPromise = null;
}

export function sourceOptions(sources, selected = '', emptyLabel = null) {
  const empty = emptyLabel === null ? '' : `<option value="">${esc(emptyLabel)}</option>`;
  return empty + sources
    .map((s) => `<option value="${esc(s.code)}" ${s.code === selected ? 'selected' : ''}>${esc(s.name)}</option>`)
    .join('');
}

export function ownerOptions(employees, selected = '', emptyLabel = null) {
  const empty = emptyLabel === null ? '' : `<option value="">${esc(emptyLabel)}</option>`;
  return empty + employees
    .map((e) => `<option value="${esc(e.id)}" ${e.id === selected ? 'selected' : ''}>${esc(e.full_name)}</option>`)
    .join('');
}

// ---------- Уведомления ----------

export function toast(message, { kind = 'info', detail = '' } = {}) {
  const box = document.getElementById('toasts');
  const el = document.createElement('div');
  el.className = `toast ${kind}`;
  el.innerHTML = `<div>${esc(message)}</div>${detail ? `<div class="toast__detail">${esc(detail)}</div>` : ''}`;
  box.append(el);
  setTimeout(() => el.remove(), kind === 'error' ? 7000 : 3500);
}

// errorText переводит ошибку API в понятную фразу и подробность для пользователя.
export function errorText(err) {
  if (!(err instanceof ApiError)) return { message: 'Что-то пошло не так', detail: String(err) };

  const titles = {
    0: 'Сервер недоступен',
    400: 'Данные не прошли проверку',
    404: 'Не найдено',
    409: 'Конфликт',
    500: 'Ошибка на сервере',
  };
  const message = titles[err.status] ?? `Ошибка ${err.status}`;
  const detail = err.status >= 500 && err.requestId ? `Код запроса: ${err.requestId}` : err.message;
  return { message, detail };
}

export function toastError(err) {
  const { message, detail } = errorText(err);
  toast(message, { kind: 'error', detail });
}

// ---------- Состояния страницы ----------

export function loading() {
  return '<div class="loading">Загрузка…</div>';
}

export function errorState(err) {
  const { message, detail } = errorText(err);
  return `<div class="card error-state">
    <div class="error-state__title">${esc(message)}</div>
    <div class="muted">${esc(detail)}</div>
  </div>`;
}

// ---------- Диалог ----------

// openDialog показывает модальное окно и возвращает элемент <dialog>; при закрытии он удаляется.
export function openDialog(html) {
  const dialog = document.createElement('dialog');
  dialog.innerHTML = html;
  document.body.append(dialog);
  dialog.addEventListener('close', () => dialog.remove());
  dialog.querySelectorAll('[data-close]').forEach((btn) => btn.addEventListener('click', () => dialog.close()));
  dialog.showModal();
  return dialog;
}

// emptyToNull превращает пустую строку из поля формы в null — «нет значения» для API.
export function emptyToNull(value) {
  const v = (value ?? '').trim();
  return v === '' ? null : v;
}
