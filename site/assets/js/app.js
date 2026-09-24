// ============================================================================
//  Точка входа: hash-роутинг, инициализация ролей и навигации
// ============================================================================
import { initAuth, setRole, getRole, onRoleChange, ROLES } from './auth.js';
import { renderImportView } from './views/importView.js';
import { renderLeadsView }  from './views/leadsView.js';
import { renderKanbanView } from './views/kanbanView.js';
import { renderReportView } from './views/reportView.js';

const ROUTES = {
  import: renderImportView,
  leads:  renderLeadsView,
  kanban: renderKanbanView,
  report: renderReportView,
};
const DEFAULT_ROUTE = 'import';

function currentRoute() {
  const hash = window.location.hash.replace(/^#\/?/, '');
  return ROUTES[hash] ? hash : DEFAULT_ROUTE;
}

function highlightNav(route) {
  document.querySelectorAll('.app-nav__link').forEach((a) => {
    a.classList.toggle('is-active', a.dataset.route === route);
  });
}

function highlightRoles() {
  const role = getRole();
  document.querySelectorAll('.role-switcher__btn').forEach((b) => {
    b.classList.toggle('is-active', b.dataset.role === role);
  });
}

function render() {
  const route = currentRoute();
  highlightNav(route);
  const root = document.getElementById('appMain');
  Promise.resolve(ROUTES[route](root)).catch((e) => {
    root.innerHTML = `<div class="empty">Ошибка экрана: ${e.message}</div>`;
    console.error(e);
  });
}

function init() {
  initAuth();
  highlightRoles();

  // переключатель ролей
  document.getElementById('roleSwitcher').addEventListener('click', (e) => {
    const btn = e.target.closest('[data-role]');
    if (!btn) return;
    setRole(btn.dataset.role);
  });

  onRoleChange(() => {
    highlightRoles();
    // если в этот момент была открыта модалка карточки — закрываем,
    // т.к. права (просмотр / редактирование) изменились
    const dealModal = document.getElementById('dealModal');
    if (dealModal && !dealModal.hidden) {
      dealModal.hidden = true;
      document.getElementById('dealModalBody').innerHTML = '';
      document.getElementById('dealModalFooter').innerHTML = '';
    }
    render(); // перерисуем текущий экран, чтобы права обновились
  });

  // первичный рендер
  if (!window.location.hash) window.location.hash = '#/' + DEFAULT_ROUTE;
  render();
  window.addEventListener('hashchange', render);

  console.info(
    `[CRM] стартовали. Роль: ${ROLES[getRole()].title}. ` +
    `Режим API: ${window.__api?.config.apiMode}. ` +
    `BASE_URL_1C: ${window.__api?.config.BASE_URL_1C}`
  );
}

document.addEventListener('DOMContentLoaded', init);
