// Страница «Импорт»: загрузка выгрузки лидов из 1С.

import { api } from '../api.js';
import { esc, errorText, resetRefs } from '../ui.js';

export function renderImport(root) {
  root.innerHTML = `
    <div class="page-head">
      <div>
        <h1>Импорт</h1>
        <div class="page-head__sub muted">Загрузка выгрузки лидов из 1С в формате JSON</div>
      </div>
    </div>

    <div class="card"><div class="card__body">
      <div class="dropzone" id="dropzone">
        <div class="dropzone__title">Перетащите файл сюда</div>
        <div class="muted">или</div>
        <div class="form-actions form-actions--center">
          <label class="btn btn--primary">
            Выбрать файл
            <input type="file" id="file" accept=".json,application/json" hidden>
          </label>
        </div>
      </div>
      <div id="result"></div>
      <p class="field__hint note">
        Повторная загрузка не создаёт дублей: сервер узнаёт тот же файл по SHA-256, а каждого лида — по номеру из 1С.
        Весь файл загружается одной транзакцией: либо все лиды, либо ни одного.
        Пример выгрузки — <span class="mono">testdata/seed_leads.json</span> в репозитории.
      </p>
    </div></div>`;

  const zone = root.querySelector('#dropzone');
  const input = root.querySelector('#file');
  const result = root.querySelector('#result');

  async function upload(file) {
    if (!file) return;
    result.className = 'result';
    result.textContent = `Загружаю ${file.name}…`;

    try {
      const res = await api.importLeads(file);
      resetRefs();
      result.className = 'result ok';
      result.innerHTML = `
        <b>Готово.</b> Загружено лидов: ${esc(res.imported)}, пропущено — уже были в базе: ${esc(res.skipped)}.
        <a href="#/leads">Открыть лидов</a> · <a href="#/report">Открыть отчёт</a>`;
    } catch (err) {
      result.className = 'result error';
      if (err.status === 409) {
        result.textContent = 'Этот файл уже загружали — повторная загрузка ничего не изменит.';
        return;
      }
      const { message, detail } = errorText(err);
      result.textContent = `${message}: ${detail}`;
    } finally {
      input.value = '';
    }
  }

  input.addEventListener('change', () => upload(input.files[0]));

  zone.addEventListener('dragover', (e) => {
    e.preventDefault();
    zone.classList.add('drop-target');
  });
  zone.addEventListener('dragleave', () => zone.classList.remove('drop-target'));
  zone.addEventListener('drop', (e) => {
    e.preventDefault();
    zone.classList.remove('drop-target');
    upload(e.dataTransfer.files[0]);
  });
}
