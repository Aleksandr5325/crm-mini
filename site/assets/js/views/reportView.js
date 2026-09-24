// ============================================================================
//  Экран: Отчёт «Воронка продаж»
// ============================================================================
//  Закрывает критерии приёмки кейса №3:
//   • №2 — конверсия по этапам И по источникам;
//   • №3 — экспорт CSV (lead_id, stage, duration_days, source) скачиванием
//          в браузер;
//   • плюс средняя длительность каждой стадии (KPI длительности).
//
//  Данные берутся с того же Go-бэка: список лидов + история стадий каждого
//  лида (GET /leads/{id}/stages). Длительности считаются по истории:
//  duration = (left_at − entered_at). Для открытой стадии — до текущего момента.
//
//  В mock-режиме история стадий пустая (нет lead_stages), поэтому отчёт нужно
//  смотреть в real-режиме (с импортированными данными).
// ============================================================================
import { h, formatDate, toast } from '../utils.js';
import { getLeads, getLeadStages, getDictionaries } from '../api.js';
import { getStagesOrdered } from '../config.js';

const DAY_MS = 86400000;

export async function renderReportView(root) {
  root.innerHTML = '';
  root.append(h('h1', { class: 'view-title' }, ['Отчёт «Воронка продаж»']));

  const info = h('p', { class: 'muted', style: 'margin: 0 0 12px;' }, [
    'Конверсия по этапам и источникам, средняя длительность стадий. ',
    'Длительности считаются по истории стадий (real-режим).',
  ]);
  const content = h('div', {}, [h('div', { class: 'empty' }, ['Считаем отчёт…'])]);
  root.append(info, content);

  try {
    const stages = getStagesOrdered();
    const [leadsResp, dicts] = await Promise.all([getLeads(), getDictionaries()]);
    const leads = leadsResp.items || [];
    const sourcesMap = Object.fromEntries((dicts.sources || []).map((s) => [s.code, s.name]));

    // Тянем историю стадий по каждому лиду (одноразовое действие для отчёта).
    const histories = await Promise.all(
      leads.map((l) =>
        getLeadStages(l.lead_id)
          .then((h) => ({ lead: l, stages: Array.isArray(h) ? h : [] }))
          .catch(() => ({ lead: l, stages: [] }))
      )
    );

    const model = buildModel(stages, leads, histories);
    content.innerHTML = '';
    content.append(
      buildExportBar(histories, sourcesMap),
      buildFunnelTable(stages, model.overall),
      buildDurationTable(stages, model.avgDurationDays),
      buildBySourceTable(stages, model.bySource, sourcesMap),
    );
  } catch (e) {
    content.innerHTML = '';
    content.append(h('div', { class: 'empty' }, ['Не удалось построить отчёт: ' + e.message]));
    toast(e.message, 'error', 5000);
  }
}

// ---------------------------------------------------------------------------
//  Расчёт модели отчёта
// ---------------------------------------------------------------------------
//  «Лид достиг стадии X», если в его истории есть запись со стадией X
//  (а если истории нет — берём текущую стадию как достигнутую).
function reachedStages(entry) {
  const set = new Set();
  for (const s of entry.stages) if (s.stage) set.add(s.stage);
  if (set.size === 0 && entry.lead.current_stage) set.add(entry.lead.current_stage);
  return set;
}

function buildModel(stages, leads, histories) {
  const codes = stages.map((s) => s.code);

  // Счётчики «достигли стадии» — всего и по источникам.
  const overallReached = Object.fromEntries(codes.map((c) => [c, 0]));
  const bySourceReached = {}; // source -> { total, reached: {code: n} }

  // Накопители длительностей по стадиям.
  const durSum = Object.fromEntries(codes.map((c) => [c, 0]));
  const durCnt = Object.fromEntries(codes.map((c) => [c, 0]));
  const now = Date.now();

  for (const entry of histories) {
    const src = entry.lead.source || '—';
    if (!bySourceReached[src]) {
      bySourceReached[src] = { total: 0, reached: Object.fromEntries(codes.map((c) => [c, 0])) };
    }
    bySourceReached[src].total += 1;

    const reached = reachedStages(entry);
    for (const c of codes) {
      if (reached.has(c)) {
        overallReached[c] += 1;
        bySourceReached[src].reached[c] += 1;
      }
    }

    // Длительности по записям истории.
    for (const st of entry.stages) {
      if (!st.stage || durSum[st.stage] === undefined) continue;
      const start = parseDate(st.entered_at);
      if (!start) continue;
      const end = parseDate(st.left_at) || new Date(now);
      const days = (end.getTime() - start.getTime()) / DAY_MS;
      if (days >= 0) {
        durSum[st.stage] += days;
        durCnt[st.stage] += 1;
      }
    }
  }

  const avgDurationDays = Object.fromEntries(
    codes.map((c) => [c, durCnt[c] ? durSum[c] / durCnt[c] : null])
  );

  return { overall: overallReached, bySource: bySourceReached, avgDurationDays };
}

function parseDate(s) {
  if (!s) return null;
  const d = new Date(String(s).replace(' ', 'T'));
  return Number.isNaN(d.getTime()) ? null : d;
}

// Конверсия стадии = достигли стадии / достигли первой стадии (new), в %.
function conversionPct(reached, codes) {
  const base = reached[codes[0]] || 0;
  if (!base) return codes.map(() => null);
  return codes.map((c) => (reached[c] / base) * 100);
}

// ---------------------------------------------------------------------------
//  Таблицы
// ---------------------------------------------------------------------------
function buildFunnelTable(stages, overall) {
  const codes = stages.map((s) => s.code);
  const conv = conversionPct(overall, codes);
  const rows = stages.map((s, i) =>
    h('tr', {}, [
      h('td', {}, [s.title]),
      h('td', {}, [String(overall[s.code] || 0)]),
      h('td', {}, [conv[i] === null ? '—' : conv[i].toFixed(1) + '%']),
    ])
  );
  return h('div', { class: 'card' }, [
    h('h2', { class: 'view-title', style: 'font-size: 18px;' }, ['Конверсия по этапам']),
    h('table', { class: 'table' }, [
      h('thead', {}, [h('tr', {}, [
        h('th', {}, ['Стадия']),
        h('th', {}, ['Достигли']),
        h('th', {}, ['Конверсия от «Новый»']),
      ])]),
      h('tbody', {}, rows),
    ]),
  ]);
}

function buildDurationTable(stages, avg) {
  const rows = stages.map((s) =>
    h('tr', {}, [
      h('td', {}, [s.title]),
      h('td', {}, [avg[s.code] === null ? '—' : avg[s.code].toFixed(1) + ' дн']),
    ])
  );
  return h('div', { class: 'card' }, [
    h('h2', { class: 'view-title', style: 'font-size: 18px;' }, ['Средняя длительность стадии']),
    h('table', { class: 'table' }, [
      h('thead', {}, [h('tr', {}, [h('th', {}, ['Стадия']), h('th', {}, ['Средняя длительность'])])]),
      h('tbody', {}, rows),
    ]),
  ]);
}

function buildBySourceTable(stages, bySource, sourcesMap) {
  const codes = stages.map((s) => s.code);
  const sources = Object.keys(bySource).sort();

  const head = h('tr', {}, [
    h('th', {}, ['Источник']),
    h('th', {}, ['Лидов']),
    ...stages.map((s) => h('th', {}, [s.title])),
  ]);

  const rows = sources.map((src) => {
    const data = bySource[src];
    const conv = conversionPct(data.reached, codes);
    return h('tr', {}, [
      h('td', {}, [sourcesMap[src] || src]),
      h('td', {}, [String(data.total)]),
      ...stages.map((s, i) =>
        h('td', {}, [conv[i] === null ? '—' : conv[i].toFixed(0) + '%'])
      ),
    ]);
  });

  return h('div', { class: 'card' }, [
    h('h2', { class: 'view-title', style: 'font-size: 18px;' }, ['Конверсия по источникам']),
    h('p', { class: 'muted', style: 'margin: 0 0 8px; font-size: 12px;' }, [
      'В ячейках — доля лидов источника, достигших стадии (от числа лидов источника).',
    ]),
    h('table', { class: 'table' }, [
      h('thead', {}, [head]),
      h('tbody', {}, rows),
    ]),
  ]);
}

// ---------------------------------------------------------------------------
//  Экспорт CSV: lead_id, stage, duration_days, source
// ---------------------------------------------------------------------------
function buildExportBar(histories, sourcesMap) {
  const btn = h('button', { class: 'btn btn--primary' }, ['Экспорт CSV']);
  btn.addEventListener('click', () => exportCsv(histories));
  return h('div', { class: 'toolbar card' }, [
    h('span', { class: 'muted' }, ['Выгрузка длительностей по стадиям для внешней аналитики/BI.']),
    btn,
  ]);
}

function csvCell(v) {
  const s = String(v ?? '');
  return /[",;\n]/.test(s) ? '"' + s.replace(/"/g, '""') + '"' : s;
}

function exportCsv(histories) {
  const now = Date.now();
  const header = ['lead_id', 'stage', 'duration_days', 'source'];
  const lines = [header.join(',')];

  for (const entry of histories) {
    const src = entry.lead.source || '';
    for (const st of entry.stages) {
      if (!st.stage) continue;
      const start = parseDate(st.entered_at);
      if (!start) continue;
      const end = parseDate(st.left_at) || new Date(now);
      const days = Math.max(0, (end.getTime() - start.getTime()) / DAY_MS);
      lines.push([
        csvCell(entry.lead.lead_id),
        csvCell(st.stage),
        csvCell(days.toFixed(2)),
        csvCell(src),
      ].join(','));
    }
  }

  if (lines.length === 1) {
    toast('Нет данных истории стадий для выгрузки (нужен real-режим с импортом).', 'error', 4000);
    return;
  }

  // BOM, чтобы Excel корректно открыл кириллицу в UTF-8.
  const blob = new Blob(['\uFEFF' + lines.join('\r\n')], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = h('a', {
    href: url,
    download: `funnel_export_${new Date().toISOString().slice(0, 10)}.csv`,
  });
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
  toast('CSV выгружен', 'success');
}
