// Страница «Источники»: справочник источников лидов и добавление нового.

import { api } from '../api.js';
import { esc, loading, errorState, errorText, toast, resetRefs } from '../ui.js';

export async function renderSources(root, ctx) {
  root.innerHTML = loading();

  let sources;
  try {
    sources = await api.sources();
  } catch (err) {
    if (ctx.alive()) root.innerHTML = errorState(err);
    return;
  }
  if (!ctx.alive()) return;

  const rows = sources.map((s) => `
    <tr><td class="mono">${esc(s.code)}</td><td>${esc(s.name)}</td></tr>`).join('');

  root.innerHTML = `
    <div class="page-head">
      <div>
        <h1>Источники</h1>
        <div class="page-head__sub muted">Откуда приходят лиды: сайт, олимпиада, рекомендация и т.п.</div>
      </div>
    </div>

    <div class="grid-2">
      <div class="card">
        ${sources.length === 0
          ? '<div class="empty">Источников пока нет.</div>'
          : `<table class="table"><thead><tr><th>Код</th><th>Название</th></tr></thead><tbody>${rows}</tbody></table>`}
      </div>

      <div class="card"><div class="card__body">
        <h2>Новый источник</h2>
        <form id="source-form">
          <div class="stack stack--tight">
            <div class="field">
              <label for="s-code">Код</label>
              <input id="s-code" name="code" required placeholder="telegram" autocomplete="off">
              <span class="field__hint">2–50 латинских строчных букв или «_». Код нельзя занять дважды.</span>
            </div>
            <div class="field">
              <label for="s-name">Название</label>
              <input id="s-name" name="name" required maxlength="100" placeholder="Телеграм" autocomplete="off">
            </div>
          </div>
          <div class="form-error" id="form-error" hidden></div>
          <div class="form-actions">
            <button class="btn btn--primary" type="submit">Добавить</button>
          </div>
        </form>
      </div></div>
    </div>`;

  const form = root.querySelector('#source-form');
  const errorBox = root.querySelector('#form-error');

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const data = new FormData(form);
    errorBox.hidden = true;

    try {
      const created = await api.createSource({ code: data.get('code'), name: data.get('name') });
      resetRefs();
      toast(`Источник «${created.name}» добавлен`, { kind: 'ok' });
      renderSources(root, ctx);
    } catch (err) {
      const { message, detail } = errorText(err);
      errorBox.textContent = `${message}: ${detail}`;
      errorBox.hidden = false;
    }
  });
}
