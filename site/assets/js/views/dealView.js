// ============================================================================
//  Модалка карточки сделки/клиента
// ============================================================================
//  Поля бэка: lead_id, source, current_stage, owner, created_at.
//  Расширенные (full_name, phone, email, comment) приходят, если их добавит
//  бэкендер; на стороне фронта они опциональны.
//
//  История стадий тянется из GET /leads/{id}/stages (см. lead.go).
//
//  Режимы:
//   - viewer / worker — только просмотр + история;
//   - manager         — то же + редактирование (требует PATCH /leads/{id}
//                       на бэке; в mock-режиме сохраняем в localStorage).
// ============================================================================
import { h, formatDate, toast } from '../utils.js';
import { getLead, updateLead, createLead, getStages, getDictionaries, getLeadStages } from '../api.js';
import { getStageByCode } from '../config.js';
import { can } from '../auth.js';

const modal = () => document.getElementById('dealModal');
const body  = () => document.getElementById('dealModalBody');
const foot  = () => document.getElementById('dealModalFooter');
const title = () => document.getElementById('dealModalTitle');

let _onSaved = null;

function close() {
  modal().hidden = true;
  body().innerHTML = '';
  foot().innerHTML = '';
  _onSaved = null;
}

document.addEventListener('click', (e) => {
  if (e.target.matches('[data-modal-close]')) close();
});
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && !modal().hidden) close();
});

/**
 * Открыть модалку сделки.
 * @param {string} leadId
 * @param {() => void} [onSaved] вызывается после успешного сохранения
 */
/**
 * Открыть модалку создания нового лида.
 * Только для роли manager.
 */
export async function openCreateLeadModal(onCreated) {
  if (!can('create')) {
    toast('Создание лидов доступно только менеджеру', 'error');
    return;
  }
  _onSaved = onCreated;
  modal().hidden = false;
  title().textContent = 'Новый лид';
  body().innerHTML = '<div class="empty">Загрузка справочников…</div>';
  foot().innerHTML = '';

  try {
    const [stages, dicts] = await Promise.all([getStages(), getDictionaries()]);
    renderCreate(stages, dicts);
  } catch (e) {
    body().innerHTML = '';
    body().append(h('div', { class: 'empty' }, ['Ошибка: ' + e.message]));
  }
}

function renderCreate(stages, dicts) {
  body().innerHTML = '';

  const inFullName = h('input', { class: 'field__input', placeholder: 'Например, Иванов Иван Иванович' });
  const inPhone    = h('input', { class: 'field__input', placeholder: '+7...' });
  const inEmail    = h('input', { class: 'field__input', type: 'email', placeholder: 'mail@example.com' });
  const inSource   = h('select', { class: 'field__select' }, [
    h('option', { value: '' }, ['— выберите источник —']),
    ...(dicts.sources || []).map((s) => h('option', { value: s.code }, [s.name])),
  ]);
  const inOwner = h('input', { class: 'field__input', list: 'ownersList', placeholder: 'Менеджер' });
  const ownersList = h('datalist', { id: 'ownersList' },
    (dicts.owners || []).map((o) => h('option', { value: o })),
  );
  const inComment = h('textarea', { class: 'field__textarea', placeholder: 'Краткий комментарий по лиду' });

  // На создании всегда стартовая стадия — new (правила воронки: только вперёд)
  body().append(
    h('p', { class: 'muted', style: 'margin: 0 0 12px; font-size: 13px;' }, [
      'Новый лид создаётся со стадией ',
      h('strong', {}, ['«Новый»']),
      '. Двигать по воронке потом — на доске «Воронка».',
    ]),
    field('ФИО *', inFullName),
    h('div', { class: 'row' }, [
      field('Телефон', inPhone),
      field('Email', inEmail),
    ]),
    h('div', { class: 'row' }, [
      field('Источник *', inSource),
      field('Менеджер', inOwner),
    ]),
    field('Комментарий', inComment),
    ownersList,
  );

  const createBtn = h('button', { class: 'btn btn--primary' }, ['Создать']);
  createBtn.addEventListener('click', async () => {
    const fullName = inFullName.value.trim();
    if (!fullName) { toast('ФИО обязательно', 'error'); return; }
    if (!inSource.value) { toast('Источник обязателен', 'error'); return; }

    createBtn.disabled = true;
    try {
      const lead = await createLead({
        full_name: fullName,
        phone:   inPhone.value.trim(),
        email:   inEmail.value.trim(),
        source:  inSource.value,
        owner:   inOwner.value.trim(),
        comment: inComment.value,
        current_stage: 'new',
      });
      toast(`Создан лид ${lead.lead_id}`, 'success');
      _onSaved?.(lead);
      close();
    } catch (e) {
      toast('Не удалось создать: ' + e.message, 'error', 5000);
      createBtn.disabled = false;
    }
  });

  foot().append(
    h('button', { class: 'btn', onClick: close }, ['Отмена']),
    createBtn,
  );
}

export async function openDealModal(leadId, onSaved) {
  _onSaved = onSaved;
  modal().hidden = false;
  title().textContent = 'Карточка лида';
  body().innerHTML = '<div class="empty">Загрузка…</div>';
  foot().innerHTML = '';

  try {
    const [lead, stages, dicts, history] = await Promise.all([
      getLead(leadId),
      getStages(),
      getDictionaries(),
      getLeadStages(leadId).catch(() => []),
    ]);
    render(lead, stages, dicts, history);
  } catch (e) {
    body().innerHTML = '';
    body().append(h('div', { class: 'empty' }, ['Ошибка: ' + e.message]));
  }
}

function render(lead, stages, dicts, history) {
  const editable = can('edit');
  const stageInfo = getStageByCode(lead.current_stage);
  const isClosed = stageInfo && (stageInfo.kind === 'won' || stageInfo.kind === 'lost');

  title().textContent = `${lead.lead_id} — ${lead.full_name || '(без ФИО)'}`;
  body().innerHTML = '';

  if (!editable || isClosed) {
    // ----- Режим просмотра -----
    body().append(viewMode(lead, stages, history, dicts));
    if (isClosed && editable) {
      foot().append(h('div', { class: 'muted', style: 'margin-right: auto; font-size: 12px;' }, [
        '🔒 Лид в закрытой стадии — редактирование запрещено',
      ]));
    }
    foot().append(h('button', { class: 'btn', onClick: close }, ['Закрыть']));
    return;
  }

  // ----- Режим редактирования (manager + не закрыт) -----
  const inFullName = h('input', { class: 'field__input', value: lead.full_name || '' });
  const inPhone    = h('input', { class: 'field__input', value: lead.phone || '' });
  const inEmail    = h('input', { class: 'field__input', type: 'email', value: lead.email || '' });
  const inSource   = h('select', { class: 'field__select' },
    (dicts.sources || []).map((s) => h('option',
      { value: s.code, selected: lead.source === s.code }, [s.name])),
  );
  const inOwner = h('input', { class: 'field__input', list: 'ownersList',
    value: lead.owner || lead.manager_name || '' });
  const ownersList = h('datalist', { id: 'ownersList' },
    (dicts.owners || []).map((o) => h('option', { value: o })),
  );
  const inComment = h('textarea', { class: 'field__textarea' }, [lead.comment || '']);

  body().append(
    h('div', { class: 'row' }, [
      field('ID лида', h('input', { class: 'field__input', value: lead.lead_id, disabled: true })),
      field('Текущая стадия', h('input', {
        class: 'field__input',
        value: stageInfo?.title || lead.current_stage,
        disabled: true,
      })),
    ]),
    h('p', { class: 'muted', style: 'margin: -8px 0 8px; font-size: 12px;' }, [
      'Стадия меняется только перетаскиванием на доске «Воронка».',
    ]),
    field('ФИО *', inFullName),
    h('div', { class: 'row' }, [
      field('Телефон', inPhone),
      field('Email', inEmail),
    ]),
    h('div', { class: 'row' }, [
      field('Источник', inSource),
      field('Менеджер', inOwner),
    ]),
    field('Комментарий', inComment),
    ownersList,
    historyBlock(history, stages),
    h('p', { class: 'muted', style: 'margin: 0; font-size: 12px;' }, [
      `Создан ${formatDate(lead.created_at)}` +
      (lead.updated_at ? ` · Обновлён ${formatDate(lead.updated_at)}` : ''),
    ]),
  );

  const saveBtn = h('button', { class: 'btn btn--primary' }, ['Сохранить']);
  saveBtn.addEventListener('click', async () => {
    const fullName = inFullName.value.trim();
    if (!fullName) {
      toast('ФИО обязательно для заполнения', 'error');
      return;
    }
    saveBtn.disabled = true;
    const patch = {
      full_name: fullName,
      phone: inPhone.value.trim(),
      email: inEmail.value.trim(),
      source: inSource.value,
      owner: inOwner.value.trim(),
      comment: inComment.value,
    };
    try {
      await updateLead(lead.lead_id, patch);
      toast('Лид сохранён', 'success');
      _onSaved?.();
      close();
    } catch (e) {
      toast('Ошибка: ' + e.message, 'error', 5000);
      saveBtn.disabled = false;
    }
  });

  foot().append(
    h('button', { class: 'btn', onClick: close }, ['Отмена']),
    saveBtn,
  );
}

function viewMode(lead, stages, history, dicts) {
  const wrap = h('div', {});
  const stageTitle = stages.find((s) => s.code === lead.current_stage)?.title || lead.current_stage || '—';
  const sourceMap = Object.fromEntries((dicts?.sources || []).map((s) => [s.code, s.name]));
  const items = [
    ['ID', lead.lead_id],
    ['ФИО', lead.full_name],
    ['Телефон', lead.phone],
    ['Email', lead.email],
    ['Источник', sourceMap[lead.source] || lead.source],
    ['Текущая стадия', stageTitle],
    ['Менеджер', lead.owner || lead.manager_name],
    ['Создан', formatDate(lead.created_at)],
    ['Обновлён', lead.updated_at ? formatDate(lead.updated_at) : '—'],
    ['Комментарий', lead.comment],
  ];

  const dl = h('dl', {
    style: 'display: grid; grid-template-columns: 140px 1fr; gap: 8px 16px; margin: 0 0 16px;',
  });
  for (const [k, v] of items) {
    dl.append(
      h('dt', { class: 'muted' }, [k]),
      h('dd', { style: 'margin: 0;' }, [v || '—']),
    );
  }
  wrap.append(dl);
  wrap.append(historyBlock(history, stages));
  return wrap;
}

function historyBlock(history, stages) {
  const stageTitle = (code) => stages.find((s) => s.code === code)?.title || code;
  if (!history || history.length === 0) {
    return h('div', { class: 'muted', style: 'font-size: 12px;' }, ['История стадий пуста.']);
  }
  return h('div', {}, [
    h('div', { class: 'field__label', style: 'margin-bottom: 6px;' }, ['История стадий']),
    h('ul', { class: 'history' },
      history.map((s) => h('li', { class: 'history__item' }, [
        h('span', { class: 'history__stage' }, [stageTitle(s.stage)]),
        h('span', { class: 'muted' }, [
          ' · ', formatDate(s.entered_at),
          s.left_at ? ` → ${formatDate(s.left_at)}` : ' → сейчас',
        ]),
      ])),
    ),
  ]);
}

function field(label, control) {
  return h('div', { class: 'field' }, [
    h('label', { class: 'field__label' }, [label]),
    control,
  ]);
}

