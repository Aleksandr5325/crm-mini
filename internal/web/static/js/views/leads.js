// Страница «Лиды»: таблица с фильтрами и страницами, создание лида.

import { api } from '../api.js';
import {
  esc, fmt, refs, stageBadge, STAGES, sourceOptions, ownerOptions,
  loading, errorState, openDialog, emptyToNull, errorText, toast,
} from '../ui.js';

const PAGE_SIZE = 20;

export async function renderLeads(root, ctx) {
  const filters = {
    source: ctx.params.get('source') ?? '',
    owner_id: ctx.params.get('owner_id') ?? '',
    stage: ctx.params.get('stage') ?? '',
  };
  const page = Math.max(1, Number(ctx.params.get('page')) || 1);

  root.innerHTML = loading();

  let r;
  let leads;
  try {
    r = await refs();
    // Просим на одну запись больше страницы: так без отдельного подсчёта понятно, есть ли следующая.
    leads = await api.leads({ ...filters, limit: PAGE_SIZE + 1, offset: (page - 1) * PAGE_SIZE });
  } catch (err) {
    if (ctx.alive()) root.innerHTML = errorState(err);
    return;
  }
  if (!ctx.alive()) return;

  const hasNext = leads.length > PAGE_SIZE;
  leads = leads.slice(0, PAGE_SIZE);
  const filtered = filters.source || filters.owner_id || filters.stage;

  root.innerHTML = `
    <div class="page-head">
      <div>
        <h1>Лиды</h1>
        <div class="page-head__sub muted">Потенциальные клиенты и их стадии в воронке продаж</div>
      </div>
      <button class="btn btn--primary" id="new-lead">+ Новый лид</button>
    </div>

    <div class="card">
      <form class="filters" id="filters">
        <div class="field">
          <label for="f-source">Источник</label>
          <select id="f-source" name="source">${sourceOptions(r.sources, filters.source, 'Все источники')}</select>
        </div>
        <div class="field">
          <label for="f-owner">Ответственный</label>
          <select id="f-owner" name="owner_id">${ownerOptions(r.employees, filters.owner_id, 'Все менеджеры')}</select>
        </div>
        <div class="field">
          <label for="f-stage">Стадия</label>
          <select id="f-stage" name="stage">
            <option value="">Все стадии</option>
            ${STAGES.map((s) => `<option value="${s.code}" ${s.code === filters.stage ? 'selected' : ''}>${s.label}</option>`).join('')}
          </select>
        </div>
        ${filtered ? '<a class="btn btn--small" href="#/leads">Сбросить</a>' : ''}
      </form>

      ${leads.length === 0 ? emptyText(filtered) : table(leads, r)}

      <div class="pager">
        <span class="muted">Страница ${page}</span>
        <div class="pager__buttons">
          <button class="btn btn--small" id="prev" ${page <= 1 ? 'disabled' : ''}>← Назад</button>
          <button class="btn btn--small" id="next" ${hasNext ? '' : 'disabled'}>Вперёд →</button>
        </div>
      </div>
    </div>`;

  const go = (p) => {
    const q = new URLSearchParams();
    for (const [key, value] of Object.entries(filters)) if (value) q.set(key, value);
    if (p > 1) q.set('page', p);
    location.hash = `#/leads${q.toString() ? `?${q}` : ''}`;
  };

  root.querySelector('#filters').addEventListener('change', (e) => {
    filters[e.target.name] = e.target.value;
    go(1);
  });
  root.querySelector('#prev').addEventListener('click', () => go(page - 1));
  root.querySelector('#next').addEventListener('click', () => go(page + 1));
  root.querySelectorAll('tr[data-id]').forEach((tr) => {
    tr.addEventListener('click', () => { location.hash = `#/leads/${tr.dataset.id}`; });
  });
  root.querySelector('#new-lead').addEventListener('click', () => newLeadDialog(r));
}

function emptyText(filtered) {
  return filtered
    ? '<div class="empty">Под фильтр ничего не подошло.</div>'
    : '<div class="empty">Лидов пока нет. Создайте первого или загрузите выгрузку на странице «<a href="#/import">Импорт</a>».</div>';
}

function table(leads, r) {
  const rows = leads.map((l) => `
    <tr class="clickable" data-id="${esc(l.id)}">
      <td class="muted nowrap">№ ${esc(l.number)}</td>
      <td>
        <div>${esc(l.full_name)}</div>
        <div class="table__sub">${esc([l.phone, l.email].filter(Boolean).join(' · '))}</div>
      </td>
      <td>${esc(r.sourceName[l.source_code] ?? l.source_code)}</td>
      <td>${esc(r.ownerName[l.owner_id] ?? '—')}</td>
      <td>${stageBadge(l.current_stage)}</td>
      <td class="muted nowrap">${fmt.date(l.created_at)}</td>
    </tr>`).join('');

  return `
    <table class="table">
      <thead>
        <tr><th>№</th><th>Лид</th><th>Источник</th><th>Ответственный</th><th>Стадия</th><th>Создан</th></tr>
      </thead>
      <tbody>${rows}</tbody>
    </table>`;
}

function newLeadDialog(r) {
  const noOwners = r.employees.length === 0;

  const dialog = openDialog(`
    <form method="dialog" id="lead-form">
      <div class="dialog__head"><h2>Новый лид</h2></div>
      <div class="dialog__body">
        ${noOwners ? '<p class="muted">Сотрудников пока нет — они появятся после загрузки выгрузки на странице «Импорт».</p>' : ''}
        <div class="form-grid">
          <div class="field span-2">
            <label for="n-name">Имя</label>
            <input id="n-name" name="full_name" required maxlength="255" autocomplete="off">
          </div>
          <div class="field">
            <label for="n-phone">Телефон</label>
            <input id="n-phone" name="phone" placeholder="+79991234567" autocomplete="off">
          </div>
          <div class="field">
            <label for="n-email">Email</label>
            <input id="n-email" name="email" type="email" autocomplete="off">
          </div>
          <div class="field">
            <label for="n-source">Источник</label>
            <select id="n-source" name="source_code">${sourceOptions(r.sources)}</select>
          </div>
          <div class="field">
            <label for="n-owner">Ответственный</label>
            <select id="n-owner" name="owner_id">${ownerOptions(r.employees)}</select>
          </div>
          <div class="field span-2">
            <label for="n-comment">Комментарий</label>
            <textarea id="n-comment" name="comment" maxlength="2000"></textarea>
          </div>
        </div>
        <div class="form-error" id="form-error" hidden></div>
        <div class="form-actions">
          <button class="btn btn--primary" type="submit" ${noOwners ? 'disabled' : ''}>Создать</button>
          <button class="btn" type="button" data-close>Отмена</button>
          <span class="field__hint">Нужен телефон или email</span>
        </div>
      </div>
    </form>`);

  const form = dialog.querySelector('#lead-form');
  const errorBox = dialog.querySelector('#form-error');

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const data = new FormData(form);
    const button = form.querySelector('[type=submit]');
    button.disabled = true;
    errorBox.hidden = true;

    try {
      const lead = await api.createLead({
        source_code: data.get('source_code'),
        owner_id: data.get('owner_id'),
        full_name: data.get('full_name').trim(),
        phone: emptyToNull(data.get('phone')),
        email: emptyToNull(data.get('email')),
        comment: emptyToNull(data.get('comment')),
      });
      dialog.close();
      toast(`Лид № ${lead.number} создан`, { kind: 'ok' });
      location.hash = `#/leads/${lead.id}`;
    } catch (err) {
      const { message, detail } = errorText(err);
      errorBox.textContent = `${message}: ${detail}`;
      errorBox.hidden = false;
      button.disabled = false;
    }
  });
}
