// Страница «Канбан»: лиды по стадиям, смена стадии перетаскиванием карточки.

import { api } from '../api.js';
import { esc, refs, stageBadge, STAGES, stageLabel, loading, errorState, toast, toastError } from '../ui.js';

const COLUMN_LIMIT = 100;

export async function renderBoard(root, ctx) {
  root.innerHTML = loading();

  let r;
  let columns;
  try {
    r = await refs();
    columns = await Promise.all(STAGES.map((s) => api.leads({ stage: s.code, limit: COLUMN_LIMIT })));
  } catch (err) {
    if (ctx.alive()) root.innerHTML = errorState(err);
    return;
  }
  if (!ctx.alive()) return;

  const byId = new Map(columns.flat().map((l) => [l.id, l]));

  root.innerHTML = `
    <div class="page-head">
      <div>
        <h1>Канбан</h1>
        <div class="page-head__sub muted">
          Перетащите карточку в другую колонку, чтобы сменить стадию. Правила воронки проверяет сервер:
          вперёд — только на один шаг, в «Проигран» — из любой открытой стадии.
        </div>
      </div>
    </div>
    <div class="board">${STAGES.map((s, i) => column(s, columns[i], r)).join('')}</div>`;

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
  const cards = leads.map((l) => `
    <div class="kcard" draggable="true" data-id="${esc(l.id)}">
      <a class="kcard__name" href="#/leads/${esc(l.id)}">${esc(l.full_name)}</a>
      <div class="kcard__meta">№ ${esc(l.number)} · ${esc(r.sourceName[l.source_code] ?? l.source_code)}</div>
      <div class="kcard__meta">${esc(r.ownerName[l.owner_id] ?? '')}</div>
    </div>`).join('');

  const count = leads.length >= COLUMN_LIMIT ? `${COLUMN_LIMIT}+` : leads.length;
  const more = leads.length >= COLUMN_LIMIT
    ? `<div class="column__more">Показаны последние ${COLUMN_LIMIT}. Все — на странице «<a href="#/leads?stage=${stage.code}">Лиды</a>».</div>`
    : '';

  return `
    <section class="column" data-stage="${stage.code}">
      <div class="column__head">${stageBadge(stage.code)}<span class="column__count">${count}</span></div>
      <div class="column__list">${cards}${more}</div>
    </section>`;
}
