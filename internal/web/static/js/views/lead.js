// Карточка лида: изменение полей с оптимистической блокировкой, смена стадии, история, удаление.

import { api } from '../api.js';
import {
  esc, fmt, refs, stageBadge, stageLabel, sourceOptions, ownerOptions,
  loading, errorState, openDialog, emptyToNull, errorText, toast, toastError,
} from '../ui.js';

// Следующая стадия воронки — только для кнопок. Решение «можно или нельзя» принимает сервер.
const NEXT_STAGE = { new: 'qualified', qualified: 'proposal', proposal: 'won' };
const CLOSED = ['won', 'lost'];

export async function renderLead(root, ctx) {
  const id = ctx.match[1];
  root.innerHTML = loading();

  let r;
  let lead;
  let history;
  try {
    [r, lead, history] = await Promise.all([refs(), api.lead(id), api.stages(id)]);
  } catch (err) {
    if (ctx.alive()) root.innerHTML = `<p><a href="#/leads">← Все лиды</a></p>${errorState(err)}`;
    return;
  }
  if (!ctx.alive()) return;

  draw();

  function draw() {
    root.innerHTML = `
      <p><a href="#/leads">← Все лиды</a></p>

      <div class="page-head">
        <div class="lead-head">
          <h1>${esc(lead.full_name)}</h1>
          <span class="lead-head__num">№ ${esc(lead.number)}</span>
          ${stageBadge(lead.current_stage)}
        </div>
      </div>

      <div class="grid-2">
        <div class="card"><div class="card__body">
          <h2>Данные лида</h2>
          ${editForm()}
        </div></div>

        <div class="stack">
          <div class="card"><div class="card__body">
            <h2>Стадия</h2>
            ${stageActions()}
          </div></div>

          <div class="card"><div class="card__body">
            <h2>История стадий</h2>
            ${timeline()}
          </div></div>

          <div class="card"><div class="card__body">
            <h2>Сведения</h2>
            <dl class="meta">
              <dt>Номер</dt><dd>№ ${esc(lead.number)}</dd>
              <dt>Номер в 1С</dt><dd>${esc(lead.external_id ?? '—')}</dd>
              <dt>Создан</dt><dd>${fmt.dateTime(lead.created_at)}</dd>
              <dt>Изменён</dt><dd>${fmt.dateTime(lead.updated_at)}</dd>
              <dt>Версия</dt><dd>${esc(lead.version)}</dd>
              <dt>ID</dt><dd class="mono">${esc(lead.id)}</dd>
            </dl>
            <div class="form-actions">
              <button class="btn btn--danger btn--small" id="delete">Удалить лида</button>
            </div>
          </div></div>
        </div>
      </div>`;

    root.querySelector('#edit-form').addEventListener('submit', save);
    root.querySelectorAll('[data-move]').forEach((btn) => btn.addEventListener('click', () => move(btn.dataset.move)));
    root.querySelector('#delete').addEventListener('click', confirmDelete);
  }

  // showConflict показывает предупреждение, не трогая форму: набранный текст не пропадает.
  function showConflict() {
    if (!root.querySelector('.notice')) {
      root.querySelector('.grid-2').insertAdjacentHTML('beforebegin', conflictNotice());
      root.querySelector('#reload').addEventListener('click', () => renderLead(root, ctx));
    }
    root.querySelector('.notice').scrollIntoView({ behavior: 'smooth', block: 'center' });
  }

  function editForm() {
    return `
      <form id="edit-form">
        <div class="form-grid">
          <div class="field span-2">
            <label for="e-name">Имя</label>
            <input id="e-name" name="full_name" value="${esc(lead.full_name)}" required maxlength="255">
          </div>
          <div class="field">
            <label for="e-phone">Телефон</label>
            <input id="e-phone" name="phone" value="${esc(lead.phone ?? '')}" placeholder="+79991234567">
          </div>
          <div class="field">
            <label for="e-email">Email</label>
            <input id="e-email" name="email" type="email" value="${esc(lead.email ?? '')}">
          </div>
          <div class="field">
            <label for="e-source">Источник</label>
            <select id="e-source" name="source_code">${sourceOptions(r.sources, lead.source_code)}</select>
          </div>
          <div class="field">
            <label for="e-owner">Ответственный</label>
            <select id="e-owner" name="owner_id">${ownerOptions(r.employees, lead.owner_id)}</select>
          </div>
          <div class="field span-2">
            <label for="e-comment">Комментарий</label>
            <textarea id="e-comment" name="comment" maxlength="2000">${esc(lead.comment ?? '')}</textarea>
          </div>
        </div>
        <div class="form-error" id="form-error" hidden></div>
        <div class="form-actions">
          <button class="btn btn--primary" type="submit">Сохранить</button>
        </div>
        <p class="field__hint">
          Изменения защищены оптимистической блокировкой: если лида параллельно изменят в другой вкладке,
          ваше сохранение не затрёт чужие правки. Попробуйте открыть карточку в двух вкладках.
        </p>
      </form>`;
  }

  function stageActions() {
    if (CLOSED.includes(lead.current_stage)) {
      return '<p class="muted">Сделка закрыта — стадию больше не меняют.</p>';
    }
    const next = NEXT_STAGE[lead.current_stage];
    return `
      <div class="stage-actions">
        <button class="btn btn--primary btn--small" data-move="${next}">Перевести в «${esc(stageLabel(next))}»</button>
        <button class="btn btn--danger btn--small" data-move="lost">Отметить проигранным</button>
      </div>`;
  }

  function timeline() {
    const items = history.map((p) => `
      <li class="${p.left_at ? '' : 'current'}">
        ${stageBadge(p.stage)}
        <div class="timeline__when">
          ${fmt.dateTime(p.entered_at)} — ${p.left_at ? fmt.dateTime(p.left_at) : 'сейчас'}
          ${p.left_at ? ` · ${fmt.days(p.entered_at, p.left_at)} дн.` : ''}
        </div>
      </li>`).join('');
    return `<ul class="timeline">${items}</ul>`;
  }

  function conflictNotice() {
    return `
      <div class="notice">
        <div>
          <b>Лида уже изменили</b> — в другой вкладке или другой сотрудник.
          Ваши правки не сохранены, чтобы не затереть чужие: скопируйте их и загрузите актуальные данные.
        </div>
        <button class="btn btn--small" id="reload">Загрузить актуальные данные</button>
      </div>`;
  }

  async function save(e) {
    e.preventDefault();
    const form = e.target;
    const data = new FormData(form);
    const errorBox = form.querySelector('#form-error');
    errorBox.hidden = true;

    // В PATCH отправляем только изменённые поля — остальные сервер не трогает.
    const next = {
      full_name: data.get('full_name').trim(),
      phone: emptyToNull(data.get('phone')),
      email: emptyToNull(data.get('email')),
      comment: emptyToNull(data.get('comment')),
      source_code: data.get('source_code'),
      owner_id: data.get('owner_id'),
    };
    const patch = {};
    for (const [key, value] of Object.entries(next)) {
      if (value !== (lead[key] ?? null)) patch[key] = value;
    }
    if (Object.keys(patch).length === 0) {
      toast('Нет изменений');
      return;
    }

    try {
      lead = await api.updateLead(lead.id, { version: lead.version, ...patch });
      toast('Сохранено', { kind: 'ok' });
      draw();
    } catch (err) {
      if (err.status === 409) {
        showConflict();
        return;
      }
      const { message, detail } = errorText(err);
      errorBox.textContent = `${message}: ${detail}`;
      errorBox.hidden = false;
    }
  }

  async function move(target) {
    try {
      lead = await api.moveStage(lead.id, lead.version, target);
      history = await api.stages(lead.id);
      toast(`Стадия: ${stageLabel(target)}`, { kind: 'ok' });
      draw();
    } catch (err) {
      if (err.status === 409 && err.message.includes('changed by someone else')) {
        showConflict();
        return;
      }
      toastError(err);
    }
  }

  function confirmDelete() {
    const dialog = openDialog(`
      <div class="dialog__head"><h2>Удалить лида?</h2></div>
      <div class="dialog__body">
        <p class="muted">Лид пропадёт из списков и отчёта, но останется в базе вместе с историей (мягкое удаление).</p>
        <div class="form-actions">
          <button class="btn btn--danger" id="confirm">Удалить</button>
          <button class="btn" data-close>Отмена</button>
        </div>
      </div>`);

    dialog.querySelector('#confirm').addEventListener('click', async () => {
      try {
        await api.deleteLead(lead.id);
        dialog.close();
        toast(`Лид № ${lead.number} удалён`, { kind: 'ok' });
        location.hash = '#/leads';
      } catch (err) {
        dialog.close();
        toastError(err);
      }
    });
  }
}
