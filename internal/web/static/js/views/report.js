// Страница «Отчёт»: воронка продаж — сколько лидов дошло до каждой стадии, конверсия и время в стадии.

import { api } from '../api.js';
import { esc, fmt, stageBadge, stageLabel, loading, errorState } from '../ui.js';

const FUNNEL = ['new', 'qualified', 'proposal', 'won'];

export async function renderReport(root, ctx) {
  root.innerHTML = loading();

  let f;
  try {
    f = await api.funnel();
  } catch (err) {
    if (ctx.alive()) root.innerHTML = errorState(err);
    return;
  }
  if (!ctx.alive()) return;

  const head = `
    <div class="page-head">
      <div>
        <h1>Отчёт «Воронка продаж»</h1>
        <div class="page-head__sub muted">Где лиды отсеиваются и сколько времени проводят в каждой стадии</div>
      </div>
    </div>`;

  if (f.total === 0) {
    root.innerHTML = `${head}<div class="card empty">Данных пока нет. Загрузите выгрузку на странице «<a href="#/import">Импорт</a>».</div>`;
    return;
  }

  const byStage = Object.fromEntries(f.stages.map((s) => [s.stage, s]));
  const lost = byStage.lost;

  root.innerHTML = `
    ${head}

    <div class="stats">
      ${stat('Всего лидов', f.total)}
      ${stat('Выиграно', f.won)}
      ${stat('Проиграно', f.lost)}
      ${stat('Доля выигранных', f.win_rate_pct === null ? '—' : `${fmt.number(f.win_rate_pct)}%`, 'среди закрытых сделок')}
    </div>

    <div class="report-visuals"><section class="conversion-visual"><div><span class="chart-label">Путь клиента</span><h2>Каждый этап имеет значение</h2><p>Количество лидов, дошедших до стадии</p></div><div class="sculpture">${FUNNEL.map((code,i)=>`<div class="sculpture-level sculpture-${i}" style="--level-width:${f.total ? Math.max(28,100*byStage[code].reached/f.total) : 28}%"><span>${esc(stageLabel(code))}</span><b>${esc(byStage[code].reached)}</b></div>`).join('')}</div></section><section class="time-visual"><span class="chart-label">Темп работы</span><h2>Время на каждом этапе</h2><p>Средняя длительность завершённых периодов</p><div class="time-bars">${FUNNEL.slice(0,3).map((code,i)=>{const days=byStage[code].avg_days;const max=Math.max(1,...FUNNEL.slice(0,3).map(c=>byStage[c].avg_days||0));return `<div class="time-column"><strong>${days===null?'—':fmt.number(days)} <small>дн.</small></strong><div class="time-bar tb-${i}" style="height:${days===null?2:Math.max(8,days/max*140)}px"></div><span>${esc(stageLabel(code))}</span></div>`}).join('')}</div></section></div>
    <div class="card"><div class="card__body">
      <h2>Показатели по стадиям</h2>
      <div class="funnel">
        ${FUNNEL.map((code) => row(byStage[code], f.total)).join('')}
      </div>

      <h2 class="section-gap">Отсев</h2>
      <div class="funnel">${row(lost, f.total)}</div>

      <p class="field__hint note">
        Дошли — сколько лидов хоть раз были в стадии. Конверсия — доля дошедших от дошедших до предыдущей стадии.
        Среднее время считается по завершённым периодам.
      </p>
    </div></div>`;
}

function stat(label, value, hint = '') {
  return `
    <div class="card"><div class="card__body">
      <div class="stat__label">${esc(label)}</div>
      <div class="stat__value">${esc(value)}</div>
      ${hint ? `<div class="stat__hint">${esc(hint)}</div>` : ''}
    </div></div>`;
}

function row(s, total) {
  const width = total > 0 ? Math.max((s.reached / total) * 100, 3) : 0;
  const parts = [`сейчас в стадии: <b>${esc(s.current)}</b>`];
  if (s.conversion_pct !== null) parts.push(`конверсия: <b>${fmt.number(s.conversion_pct)}%</b>`);
  if (s.avg_days !== null) parts.push(`в среднем <b>${fmt.number(s.avg_days)} дн.</b>`);

  return `
    <div class="funnel__row">
      <div>${stageBadge(s.stage)}</div>
      <div class="funnel__track">
        <div class="funnel__bar bar-${esc(s.stage)}" style="width:${width}%" title="Дошли до стадии">${esc(s.reached)}</div>
      </div>
      <div class="funnel__info">${parts.join(' · ')}</div>
    </div>`;
}
