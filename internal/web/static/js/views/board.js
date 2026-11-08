// Страница «Канбан»: лиды по стадиям, смена стадии перетаскиванием карточки.

import { api } from '../api.js';
import { esc, fmt, refs, stageBadge, STAGES, stageLabel, loading, errorState, toast, toastError } from '../ui.js';

const COLUMN_LIMIT = 100;

export async function renderBoard(root, ctx) {
  root.innerHTML = loading();

  let r;
  let columns;
  let summary;
  try {
    r = await refs();
    summary = await api.funnel();
    columns = await Promise.all(STAGES.map((s) => api.leads({ stage: s.code, limit: COLUMN_LIMIT })));
  } catch (err) {
    if (ctx.alive()) root.innerHTML = errorState(err);
    return;
  }
  if (!ctx.alive()) return;

  const byId = new Map(columns.flat().map((l) => [l.id, l]));

  root.innerHTML = `
    <div class="page-head"><div><h1>Воронка продаж<span class="heading-dot">.</span></h1><div class="page-head__sub muted">Контакты, которые становятся клиентами.</div></div><button class="btn btn--primary" id="board-create">＋ Новый лид</button></div>
    <section class="sales-hero"><div class="hero-content"><span class="hero-tag"><span></span> Ваша воронка сегодня</span><h2>Большие сделки начинаются<br>с простого «здравствуйте».</h2><p>Все контакты, следующие шаги и результаты — в одном месте.</p><div class="hero-stats"><div><strong>${summary.total}</strong><span>всего лидов</span></div><div><strong>${summary.total-summary.won-summary.lost}</strong><span>в работе</span></div><div><strong>${summary.won}</strong><span>выиграно</span></div><div><strong>${summary.win_rate_pct === null ? '—' : `${fmt.number(summary.win_rate_pct)}<small>%</small>`}</strong><span>успех среди закрытых</span></div></div></div><div class="funnel-art" aria-hidden="true"><div class="orbit orbit-one"></div><div class="orbit orbit-two"></div><i class="art-star star-one">✦</i><i class="art-star star-two">✧</i><div class="glass-stack"><div class="glass glass-one"></div><div class="glass glass-two"></div><div class="glass glass-three"></div><div class="glass glass-four"></div></div><span class="art-caption">От контакта к результату</span></div></section>
    <div class="board-toolbar"><div class="view-tabs"><span class="selected">▦ Доска</span><a href="#/leads">☷ Список</a></div><span class="board-hint">Перетащите карточку на следующий этап</span><span class="board-total">${summary.total} лидов</span></div>
    <div class="board">${STAGES.map((s, i) => column(s, columns[i], r)).join('')}</div>`;
  root.querySelector('#board-create').addEventListener('click',()=>{location.hash='#/leads?create=1';});

  root.querySelectorAll('.kcard').forEach((card) => {
    card.addEventListener('dragstart', (e) => {
      e.dataTransfer.setData('text/plain', card.dataset.id);
      e.dataTransfer.effectAllowed = 'move';
      card.classList.add('dragging');
    });
    card.addEventListener('dragend', () => {
      card.classList.remove('dragging');
      root.querySelectorAll('.column').forEach((c) => c.classList.remove('drop-target'));
    });
  });

  root.querySelectorAll('.column').forEach((col) => {
    col.addEventListener('dragover', (e) => {
      e.preventDefault();
      col.classList.add('drop-target');
    });
    col.addEventListener('dragleave', (e) => {
      if (!col.contains(e.relatedTarget)) col.classList.remove('drop-target');
    });
    col.addEventListener('drop', async (e) => {
      e.preventDefault();
      col.classList.remove('drop-target');

      const lead = byId.get(e.dataTransfer.getData('text/plain'));
      const target = col.dataset.stage;
      if (!lead || lead.current_stage === target) return;

      const card = root.querySelector(`.kcard[data-id="${CSS.escape(lead.id)}"]`);
      card?.classList.add('busy');

      try {
        await api.moveStage(lead.id, lead.version, target);
        toast(`Лид № ${lead.number}: ${stageLabel(lead.current_stage)} → ${stageLabel(target)}`, { kind: 'ok' });
        renderBoard(root, ctx);
      } catch (err) {
        card?.classList.remove('busy');
        if (err.status === 409) {
          toast('Переход не выполнен', { kind: 'error', detail: err.message });
        } else {
          toastError(err);
        }
      }
    });
  });
}

function column(stage, leads, r) {
  const cards = leads.map((l, i) => `
    <div class="kcard" tabindex="0" draggable="true" data-id="${esc(l.id)}">
      <div class="card-topline"><span class="source-chip">${esc(r.sourceName[l.source_code] ?? l.source_code)}</span><span class="card-number">№ ${esc(l.number)}</span></div><a class="kcard__name" href="#/leads/${esc(l.id)}">${esc(l.full_name)}</a>
      <div class="kcard__contact">${esc(l.email || l.phone || "Контакт не указан")}</div>
      <div class="card-bottom"><span class="owner-avatar owner-${i%4}">${esc((r.ownerName[l.owner_id]||"М").slice(0,1))}</span><span>${esc(r.ownerName[l.owner_id] ?? "")}</span><span class="card-date">${fmt.date(l.created_at)}</span></div>
    </div>`).join('');

  const count = leads.length >= COLUMN_LIMIT ? `${COLUMN_LIMIT}+` : leads.length;
  const more = leads.length >= COLUMN_LIMIT
    ? `<div class="column__more">Показаны последние ${COLUMN_LIMIT}. Все — на странице «<a href="#/leads?stage=${stage.code}">Лиды</a>».</div>`
    : '';

  return `
    <section class="column col-${stage.code}" data-stage="${stage.code}">
      <div class="column__head"><span class="stage-dot"></span><span>${esc(stage.label)}</span><span class="column__count">${count}</span></div>
      <div class="column__list">${cards}${more}</div>
    </section>`;
}
