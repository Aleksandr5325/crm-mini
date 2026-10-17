// Точка входа интерфейса: маршрутизация по адресу после # и индикатор состояния сервиса.

import { api } from './api.js';
import { renderLeads } from './views/leads.js';
import { renderLead } from './views/lead.js';
import { renderBoard } from './views/board.js';
import { renderReport } from './views/report.js';
import { renderSources } from './views/sources.js';
import { renderImport } from './views/import.js';

const root = document.getElementById('app');

const routes = [
  { pattern: /^\/leads\/([0-9a-f-]{36})$/, nav: 'leads', render: renderLead },
  { pattern: /^\/leads$/, nav: 'leads', render: renderLeads },
  { pattern: /^\/board$/, nav: 'board', render: renderBoard },
  { pattern: /^\/report$/, nav: 'report', render: renderReport },
  { pattern: /^\/sources$/, nav: 'sources', render: renderSources },
  { pattern: /^\/import$/, nav: 'import', render: renderImport },
];

// Номер текущей отрисовки. Страница, которая ещё грузит данные, после ответа сервера проверяет
// ctx.alive(): если пользователь уже ушёл на другую страницу, старый ответ не затрёт новую.
let renderId = 0;

function route() {
  const hash = location.hash.slice(1) || '/leads';
  const [path, qs = ''] = hash.split('?');

  for (const r of routes) {
    const match = path.match(r.pattern);
    if (!match) continue;

    document.querySelectorAll('[data-nav]').forEach((a) => {
      a.classList.toggle('active', a.dataset.nav === r.nav);
    });

    const id = ++renderId;
    const ctx = { match, params: new URLSearchParams(qs), alive: () => id === renderId };
    r.render(root, ctx);
    window.scrollTo(0, 0);
    return;
  }

  location.hash = '#/leads';
}

async function checkHealth() {
  const el = document.getElementById('health');
  const text = el.querySelector('.health__text');
  try {
    await api.health();
    el.className = 'health ok';
    text.textContent = 'сервис работает';
  } catch (err) {
    el.className = 'health down';
    text.textContent = err.status === 503 ? 'нет связи с базой' : 'сервер недоступен';
  }
}

window.addEventListener('hashchange', route);
route();
checkHealth();
setInterval(checkHealth, 30_000);
