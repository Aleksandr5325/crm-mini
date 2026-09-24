// ============================================================================
//  Экран: канбан-воронка
// ============================================================================
//  Правила движения карточек (см. config.canMoveStage):
//    1. Нельзя двигать назад: целевая стадия должна быть строго правее текущей.
//    2. Нельзя двигать из won/lost вообще — это конечные стадии («попал в
//       закрытый — значит закрыт»).
//    3. Просмотрщик (viewer) карточки вообще не двигает.
// ============================================================================
import { h, formatDate, toast } from '../utils.js';
import { getLeads, getStages, moveLead, getDictionaries } from '../api.js';
import { canMoveStage, getStageByCode } from '../config.js';
import { can } from '../auth.js';
import { openDealModal } from './dealView.js';

let _stages = [];
let _allLeads = [];
let _filteredLeads = [];
let _sourcesMap = {}; // code → human name
let _filters = { dateFrom: '', dateTo: '', source: '', owner: '' };

export async function renderKanbanView(root) {
  root.innerHTML = '';
  root.append(h('h1', { class: 'view-title' }, ['Воронка продаж']));

  const stats = h('div', { class: 'stats' });
  const toolbar = h('div', { class: 'toolbar card' });
  const hint = h('p', { class: 'muted', style: 'margin: 0 0 12px;' });
  const board = h('div', { class: 'kanban', id: 'kanban' });

  hint.textContent = can('move')
    ? 'Перетащите карточку в нужную стадию. Назад вернуть нельзя; из «закрытых» — никуда.'
    : 'Режим просмотра. Чтобы перетаскивать карточки, переключите роль на «Работник» или «Менеджер».';

  root.append(stats, toolbar, hint, board);

  try {
    const [stages, leadsResp, dicts] = await Promise.all([
      getStages(), getLeads(), getDictionaries(),
    ]);
    _stages = [...stages].sort((a, b) => (a.order ?? 0) - (b.order ?? 0));
    _allLeads = leadsResp.items || [];
    _sourcesMap = Object.fromEntries((dicts.sources || []).map((s) => [s.code, s.name]));
    renderToolbar(toolbar, dicts);
    applyFilters();
    renderStats(stats);
    drawBoard(board);
  } catch (e) {
    board.innerHTML = '';
    board.append(h('div', { class: 'empty' }, ['Не удалось загрузить данные: ' + e.message]));
    toast(e.message, 'error', 5000);
  }
}

// ---------------------------------------------------------------------------
//  Тулбар фильтров
// ---------------------------------------------------------------------------
function renderToolbar(root, dicts) {
  root.innerHTML = '';

  const dateFrom = h('input', { type: 'date', class: 'field__input', value: _filters.dateFrom });
  const dateTo   = h('input', { type: 'date', class: 'field__input', value: _filters.dateTo });
  const sourceSel = h('select', { class: 'field__select' }, [
    h('option', { value: '' }, ['Все источники']),
    ...(dicts.sources || []).map((s) => h('option', { value: s.code }, [s.name])),
  ]);
  const ownerSel = h('select', { class: 'field__select' }, [
    h('option', { value: '' }, ['Все менеджеры']),
    ...(dicts.owners || []).map((o) => h('option', { value: o }, [o])),
  ]);
  const resetBtn = h('button', {
    class: 'btn btn--ghost',
    onClick: () => {
      _filters = { dateFrom: '', dateTo: '', source: '', owner: '' };
      dateFrom.value = ''; dateTo.value = '';
      sourceSel.value = ''; ownerSel.value = '';
      refresh();
    },
  }, ['Сбросить']);

  root.append(
    field('Период от', dateFrom),
    field('Период до', dateTo),
    field('Источник', sourceSel),
    field('Менеджер', ownerSel),
    resetBtn,
  );

  const onChange = () => {
    _filters = {
      dateFrom: dateFrom.value,
      dateTo: dateTo.value,
      source: sourceSel.value,
      owner: ownerSel.value,
    };
    refresh();
  };
  dateFrom.addEventListener('change', onChange);
  dateTo.addEventListener('change', onChange);
  sourceSel.addEventListener('change', onChange);
  ownerSel.addEventListener('change', onChange);
}

function field(label, control) {
  return h('div', { class: 'field' }, [
    h('label', { class: 'field__label' }, [label]),
    control,
  ]);
}

function applyFilters() {
  const { dateFrom, dateTo, source, owner } = _filters;
  const from = dateFrom ? new Date(dateFrom + 'T00:00:00') : null;
  const to   = dateTo   ? new Date(dateTo   + 'T23:59:59') : null;

  _filteredLeads = _allLeads.filter((l) => {
    if (source && l.source !== source) return false;
    if (owner && (l.owner || l.manager_name) !== owner) return false;
    if (from || to) {
      const created = parseDate(l.created_at);
      if (!created) return false;
      if (from && created < from) return false;
      if (to   && created > to)   return false;
    }
    return true;
  });
}

function parseDate(s) {
  if (!s) return null;
  // допускаем и "2026-03-20T10:30:00", и "2026-03-20 10:30:00"
  const d = new Date(String(s).replace(' ', 'T'));
  return Number.isNaN(d.getTime()) ? null : d;
}

// ---------------------------------------------------------------------------
//  Статистика сверху
// ---------------------------------------------------------------------------
function renderStats(root) {
  root.innerHTML = '';
  const total = _filteredLeads.length;
  const won = _filteredLeads.filter((l) => l.current_stage === 'won').length;
  const conv = total > 0 ? ((won / total) * 100).toFixed(1) : '0.0';
  const avgDays = avgWonDays(_filteredLeads);

  root.append(
    statCard('Всего лидов', total),
    statCard('Конверсия в Won', conv + '%'),
    statCard('Среднее время до Won', avgDays + ' дн'),
  );
}

function statCard(label, value) {
  return h('div', { class: 'stat-card' }, [
    h('span', { class: 'stat-card__label' }, [label]),
    h('span', { class: 'stat-card__value' }, [String(value)]),
  ]);
}

function avgWonDays(leads) {
  const wons = leads.filter((l) => l.current_stage === 'won' && l.created_at);
  if (!wons.length) return '0.0';
  const now = Date.now();
  const total = wons.reduce((acc, l) => {
    const t = parseDate(l.created_at)?.getTime();
    if (!t) return acc;
    return acc + (now - t) / 86400000;
  }, 0);
  return (total / wons.length).toFixed(1);
}

// ---------------------------------------------------------------------------
//  Доска
// ---------------------------------------------------------------------------
function drawBoard(board) {
  board.innerHTML = '';

  for (const stage of _stages) {
    const stageLeads = _filteredLeads.filter((l) => l.current_stage === stage.code);
    const list = h('div', { class: 'kanban__col-list', dataset: { stage: stage.code } });
    for (const lead of stageLeads) list.appendChild(buildCard(lead, stage));

    const col = h('div', {
      class: 'kanban__col' + (stage.kind === 'won' ? ' kanban__col--won' :
                              stage.kind === 'lost' ? ' kanban__col--lost' : ''),
      dataset: { stage: stage.code },
    }, [
      h('div', { class: 'kanban__col-head' }, [
        h('span', {}, [stage.title]),
        h('span', { class: 'kanban__count' }, [String(stageLeads.length)]),
      ]),
      list,
    ]);

    if (can('move')) {
      list.addEventListener('dragover', (e) => {
        e.preventDefault();
        const draggedId = e.dataTransfer.types.includes('text/plain')
          ? null  // нельзя достать значение в dragover — только в drop
          : null;
        // Подсветка: если карточка из той же доски и движение легально — show
        const fromStage = window.__draggedFromStage;
        if (fromStage && canMoveStage(fromStage, stage.code)) {
          col.classList.add('is-drop-target');
        } else {
          col.classList.add('is-drop-forbidden');
        }
      });
      list.addEventListener('dragleave', () => {
        col.classList.remove('is-drop-target', 'is-drop-forbidden');
      });
      list.addEventListener('drop', async (e) => {
        e.preventDefault();
        col.classList.remove('is-drop-target', 'is-drop-forbidden');
        const leadId = e.dataTransfer.getData('text/plain');
        if (!leadId) return;
        const lead = _allLeads.find((l) => String(l.lead_id) === String(leadId));
        if (!lead) return;
        const fromStage = lead.current_stage;
        const toStage = stage.code;

        if (!canMoveStage(fromStage, toStage)) {
          const from = getStageByCode(fromStage);
          if (from && (from.kind === 'won' || from.kind === 'lost')) {
            toast('Лид в закрытой стадии — его нельзя двигать', 'error', 3500);
          } else {
            toast('Нельзя вернуть лид на предыдущую стадию', 'error', 3500);
          }
          return;
        }

        // Оптимистичное обновление
        lead.current_stage = toStage;
        applyFilters();
        drawBoard(board);
        renderStats(document.querySelector('.stats'));

        try {
          await moveLead(leadId, toStage);
          toast(`Стадия: ${getStageByCode(fromStage)?.title} → ${getStageByCode(toStage)?.title}`, 'success', 2200);
        } catch (err) {
          // Откатываем
          lead.current_stage = fromStage;
          applyFilters();
          drawBoard(board);
          renderStats(document.querySelector('.stats'));
          toast('Не удалось сохранить: ' + err.message, 'error', 5000);
        }
      });
    }

    board.appendChild(col);
  }
}

function buildCard(lead, stage) {
  const locked = stage.kind === 'won' || stage.kind === 'lost' || !can('move');

  const card = h('div', {
    class: 'kanban__card' + (locked ? ' kanban__card--locked' : ''),
    draggable: locked ? 'false' : 'true',
    onClick: () => openDealModal(lead.lead_id, () => {
      refresh();
    }),
    title: locked && (stage.kind === 'won' || stage.kind === 'lost')
      ? 'Закрытая стадия — лид зафиксирован'
      : '',
  }, [
    h('div', { class: 'kanban__card-name' }, [lead.full_name || lead.lead_id]),
    h('div', { class: 'kanban__card-meta' }, [
      [lead.phone, lead.email].filter(Boolean).join(' · ') || lead.lead_id,
    ]),
    h('div', { class: 'kanban__card-meta' }, [
      [_sourcesMap[lead.source] || lead.source, lead.owner || lead.manager_name]
        .filter(Boolean).join(' · ') || '—',
    ]),
    h('div', { class: 'kanban__card-amount' }, [formatDate(lead.created_at)]),
  ]);

  if (!locked) {
    card.addEventListener('dragstart', (e) => {
      e.dataTransfer.setData('text/plain', String(lead.lead_id));
      e.dataTransfer.effectAllowed = 'move';
      window.__draggedFromStage = stage.code;
      card.classList.add('is-dragging');
    });
    card.addEventListener('dragend', () => {
      window.__draggedFromStage = null;
      card.classList.remove('is-dragging');
    });
  }
  return card;
}

// ---------------------------------------------------------------------------
//  Перерисовка всех частей
// ---------------------------------------------------------------------------
async function refresh() {
  try {
    const leadsResp = await getLeads();
    _allLeads = leadsResp.items || [];
    applyFilters();
    renderStats(document.querySelector('.stats'));
    drawBoard(document.getElementById('kanban'));
  } catch (e) {
    toast('Не удалось обновить доску: ' + e.message, 'error');
  }
}
