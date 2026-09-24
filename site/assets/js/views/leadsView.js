// ============================================================================
//  Экран: список лидов с фильтрами/поиском/сортировкой/пагинацией
// ============================================================================
//  Поля Lead — как у Go-бэка (см. lead.go): lead_id, source, current_stage,
//  owner, created_at + расширения с фронта (full_name, phone, email, comment).
// ============================================================================
import { h, debounce, formatDate, toast } from '../utils.js';
import { getLeads, getStages, getDictionaries } from '../api.js';
import { CONFIG } from '../config.js';
import { can } from '../auth.js';
import { openDealModal, openCreateLeadModal } from './dealView.js';

const state = {
  all: [],
  filtered: [],
  page: 1,
  sort: { key: 'created_at', dir: 'desc' },
  filters: { q: '', stage: '', source: '', owner: '', dateFrom: '', dateTo: '' },
};

let _stageMap = {};
let _sourceMap = {};

export async function renderLeadsView(root) {
  root.innerHTML = '';
  root.append(h('h1', { class: 'view-title' }, ['Лиды']));

  const toolbar = h('div', { class: 'toolbar card' });
  const tableWrap = h('div', { class: 'table-wrap' });

  root.append(toolbar, tableWrap);

  try {
    const [leadsResp, stages, dicts] = await Promise.all([
      getLeads(), getStages(), getDictionaries(),
    ]);
    state.all = leadsResp.items || [];
    _stageMap = Object.fromEntries(stages.map((s) => [s.code, s.title]));
    _sourceMap = Object.fromEntries((dicts.sources || []).map((s) => [s.code, s.name]));
    renderToolbar(toolbar, stages, dicts);
    applyAndRender(tableWrap);
  } catch (e) {
    tableWrap.innerHTML = '';
    tableWrap.append(h('div', { class: 'empty' }, [
      'Ошибка загрузки данных: ' + e.message,
    ]));
    toast(e.message, 'error', 5000);
  }
}

function renderToolbar(root, stages, dicts) {
  root.innerHTML = '';

  const qInput = h('input', {
    type: 'search',
    class: 'field__input',
    placeholder: 'Поиск: ID, ФИО, email, телефон',
    style: 'min-width: 240px;',
  });
  const stageSelect = h('select', { class: 'field__select' }, [
    h('option', { value: '' }, ['Все стадии']),
    ...stages.map((s) => h('option', { value: s.code }, [s.title])),
  ]);
  const sourceSelect = h('select', { class: 'field__select' }, [
    h('option', { value: '' }, ['Все источники']),
    ...(dicts.sources || []).map((s) => h('option', { value: s.code }, [s.name])),
  ]);
  const ownerSelect = h('select', { class: 'field__select' }, [
    h('option', { value: '' }, ['Все менеджеры']),
    ...(dicts.owners || []).map((o) => h('option', { value: o }, [o])),
  ]);
  const dateFromInput = h('input', { type: 'date', class: 'field__input' });
  const dateToInput   = h('input', { type: 'date', class: 'field__input' });

  const resetBtn = h('button', {
    class: 'btn btn--ghost',
    onClick: () => {
      state.filters = { q: '', stage: '', source: '', owner: '', dateFrom: '', dateTo: '' };
      qInput.value = ''; stageSelect.value = '';
      sourceSelect.value = ''; ownerSelect.value = '';
      dateFromInput.value = ''; dateToInput.value = '';
      state.page = 1;
      applyAndRender(document.querySelector('.table-wrap'));
    },
  }, ['Сбросить']);

  // Кнопка создания — только менеджеру
  const createBtn = h('button', {
    class: 'btn btn--primary',
    title: can('create') ? '' : 'Создание доступно только менеджеру',
    disabled: !can('create'),
    onClick: async () => {
      await openCreateLeadModal(async (newLead) => {
        // Перечитываем список с бэка/мока и перерисуем таблицу
        try {
          const r = await getLeads();
          state.all = r.items || [];
          applyAndRender(document.querySelector('.table-wrap'));
        } catch (e) {
          toast('Создан, но не удалось обновить список: ' + e.message, 'warn');
        }
      });
    },
  }, ['+ Новый лид']);

  root.append(
    field('Поиск', qInput),
    field('Период от', dateFromInput),
    field('Период до', dateToInput),
    field('Стадия', stageSelect),
    field('Источник', sourceSelect),
    field('Менеджер', ownerSelect),
    resetBtn,
    createBtn,
  );

  const onChange = () => {
    state.filters = {
      q: qInput.value.trim(),
      stage: stageSelect.value,
      source: sourceSelect.value,
      owner: ownerSelect.value,
      dateFrom: dateFromInput.value,
      dateTo: dateToInput.value,
    };
    state.page = 1;
    applyAndRender(document.querySelector('.table-wrap'));
  };
  qInput.addEventListener('input', debounce(onChange, 200));
  stageSelect.addEventListener('change', onChange);
  sourceSelect.addEventListener('change', onChange);
  ownerSelect.addEventListener('change', onChange);
  dateFromInput.addEventListener('change', onChange);
  dateToInput.addEventListener('change', onChange);
}

function field(label, control) {
  return h('div', { class: 'field' }, [
    h('label', { class: 'field__label' }, [label]),
    control,
  ]);
}

function parseDate(s) {
  if (!s) return null;
  const d = new Date(String(s).replace(' ', 'T'));
  return Number.isNaN(d.getTime()) ? null : d;
}

function applyAndRender(tableWrap) {
  const { q, stage, source, owner, dateFrom, dateTo } = state.filters;
  const ql = q.toLowerCase();
  const from = dateFrom ? new Date(dateFrom + 'T00:00:00') : null;
  const to   = dateTo   ? new Date(dateTo   + 'T23:59:59') : null;

  state.filtered = state.all.filter((l) => {
    if (stage  && l.current_stage !== stage) return false;
    if (source && l.source !== source) return false;
    const own = l.owner || l.manager_name;
    if (owner  && own !== owner) return false;
    if (from || to) {
      const c = parseDate(l.created_at);
      if (!c) return false;
      if (from && c < from) return false;
      if (to   && c > to)   return false;
    }
    if (q) {
      const hay = [l.lead_id, l.full_name, l.email, l.phone]
        .map((x) => (x || '').toLowerCase()).join(' ');
      if (!hay.includes(ql)) return false;
    }
    return true;
  });

  // сортировка
  const { key, dir } = state.sort;
  const sign = dir === 'asc' ? 1 : -1;
  state.filtered.sort((a, b) => {
    const av = a[key]; const bv = b[key];
    if (av === bv) return 0;
    if (av === undefined || av === null) return 1;
    if (bv === undefined || bv === null) return -1;
    if (typeof av === 'number' && typeof bv === 'number') return (av - bv) * sign;
    return String(av).localeCompare(String(bv), 'ru') * sign;
  });

  renderTable(tableWrap);
}

function renderTable(root) {
  root.innerHTML = '';

  const cols = [
    { key: 'lead_id',       title: 'ID' },
    { key: 'full_name',     title: 'ФИО' },
    { key: 'phone',         title: 'Телефон' },
    { key: 'email',         title: 'Email' },
    { key: 'source',        title: 'Источник', format: (v) => _sourceMap[v] || v || '—' },
    { key: 'current_stage', title: 'Стадия' },
    { key: 'owner',         title: 'Менеджер', format: (v, l) => v || l.manager_name || '—' },
    { key: 'created_at',    title: 'Создан', format: formatDate },
  ];

  const headRow = h('tr', {}, cols.map((c) => {
    const isSorted = state.sort.key === c.key;
    const arrow = isSorted ? (state.sort.dir === 'asc' ? '▲' : '▼') : '⇅';
    return h('th', {
      onClick: () => {
        if (state.sort.key === c.key) {
          state.sort.dir = state.sort.dir === 'asc' ? 'desc' : 'asc';
        } else {
          state.sort.key = c.key; state.sort.dir = 'asc';
        }
        applyAndRender(root);
      },
    }, [c.title, h('span', { class: 'sort' }, [arrow])]);
  }));

  const total = state.filtered.length;
  const totalPages = Math.max(1, Math.ceil(total / CONFIG.pageSize));
  if (state.page > totalPages) state.page = totalPages;
  const start = (state.page - 1) * CONFIG.pageSize;
  const pageItems = state.filtered.slice(start, start + CONFIG.pageSize);

  const body = pageItems.length === 0
    ? h('tr', {}, [h('td', { colspan: cols.length, class: 'empty' }, ['Ничего не найдено'])])
    : pageItems.map((l) => h('tr', {
        onClick: () => openDealModal(l.lead_id, () => applyAndRender(root)),
      }, cols.map((c) => {
        const raw = l[c.key];
        if (c.key === 'current_stage') {
          const cls = raw === 'won' ? 'badge--won' : raw === 'lost' ? 'badge--lost' : '';
          return h('td', {}, [h('span', { class: `badge ${cls}` }, [_stageMap[raw] || raw || '—'])]);
        }
        const val = c.format ? c.format(raw, l) : (raw ?? '—');
        return h('td', {}, [val || '—']);
      })));

  const table = h('table', { class: 'table' }, [
    h('thead', {}, [headRow]),
    h('tbody', {}, body),
  ]);

  const prevBtn = h('button', {
    class: 'btn', disabled: state.page <= 1,
    onClick: () => { state.page--; renderTable(root); },
  }, ['← Назад']);
  const nextBtn = h('button', {
    class: 'btn', disabled: state.page >= totalPages,
    onClick: () => { state.page++; renderTable(root); },
  }, ['Вперёд →']);

  const pagination = h('div', { class: 'pagination' }, [
    h('span', { class: 'pagination__info' }, [
      `Найдено: ${total}. Страница ${state.page} из ${totalPages}.`,
    ]),
    prevBtn, nextBtn,
  ]);

  root.append(table, pagination);
}
