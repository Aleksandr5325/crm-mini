// ============================================================================
//  Экран: импорт CSV
// ============================================================================
import { h, parseCsv, toast, escapeHtml } from '../utils.js';
import { uploadCsv } from '../api.js';
import { CONFIG } from '../config.js';
import { can } from '../auth.js';

export function renderImportView(root) {
  root.innerHTML = '';

  const title = h('h1', { class: 'view-title' }, ['Импорт CSV']);

  // <label> сам открывает file picker при клике на его содержимое —
  // отдельный onClick не нужен (иначе диалог открывается дважды).
  const dropzone = h('label', {
    class: 'dropzone', id: 'dropzone',
  }, [
    h('div', {}, [
      h('strong', {}, ['Перетащите файл сюда или нажмите, чтобы выбрать']),
      h('p', { class: 'muted', style: 'margin: 6px 0 0' }, [
        'Принимаются файлы .csv (UTF-8 или Windows-1251). ',
        'Разделитель определится автоматически (запятая, точка с запятой или табуляция).',
      ]),
    ]),
  ]);
  const fileInput = h('input', {
    type: 'file', accept: '.csv,text/csv,application/vnd.ms-excel',
  });
  dropzone.appendChild(fileInput);

  const fileInfo = h('div', { id: 'fileInfo', class: 'muted', style: 'margin-top: 10px;' });
  const previewWrap = h('div', { id: 'previewWrap' });
  const progress = h('div', { class: 'progress', hidden: true }, [
    h('div', { class: 'progress__bar', id: 'progressBar' }),
  ]);
  const sendBtn = h('button', {
    class: 'btn btn--primary',
    style: 'margin-top: 14px;',
    disabled: true,
    onClick: doSend,
  }, ['Отправить в 1С']);

  const log = h('pre', { class: 'log', id: 'log', hidden: true });

  const card = h('div', { class: 'card' }, [
    dropzone, fileInfo, previewWrap, progress, sendBtn, log,
  ]);

  // === Подсказка для роли без прав ===
  if (!can('import')) {
    const warn = h('div', { class: 'card', style: 'border-color: #f0c200; background: #fffaea;' }, [
      h('strong', {}, ['Импорт доступен только менеджеру.']),
      h('p', { class: 'muted', style: 'margin: 6px 0 0' }, [
        'Переключите роль в правом верхнем углу. Сейчас текущая роль не может отправлять файлы.',
      ]),
    ]);
    root.append(title, warn, card);
    sendBtn.disabled = true;
    sendBtn.title = 'Недостаточно прав';
  } else {
    root.append(title, card);
  }

  // ----------------- DnD ------------------
  let chosenFile = null;

  ['dragenter', 'dragover'].forEach((evt) =>
    dropzone.addEventListener(evt, (e) => {
      e.preventDefault();
      dropzone.classList.add('is-drag');
    }));
  ['dragleave', 'drop'].forEach((evt) =>
    dropzone.addEventListener(evt, (e) => {
      e.preventDefault();
      dropzone.classList.remove('is-drag');
    }));
  dropzone.addEventListener('drop', (e) => {
    const f = e.dataTransfer?.files?.[0];
    if (f) handleFile(f);
  });
  fileInput.addEventListener('change', () => {
    const f = fileInput.files?.[0];
    if (f) handleFile(f);
  });

  async function handleFile(file) {
    chosenFile = file;
    fileInfo.textContent = `Выбран: ${file.name} (${(file.size / 1024).toFixed(1)} КБ)`;
    sendBtn.disabled = !can('import');

    // Превью
    try {
      // Пробуем UTF-8, если получим много  — пробуем windows-1251
      let text = await file.text();
      if (/�/.test(text)) {
        // некорректные символы — перечитываем как 1251
        const buf = await file.arrayBuffer();
        text = new TextDecoder('windows-1251').decode(buf);
      }
      const { delimiter, rows } = parseCsv(text);
      const head = rows[0] || [];
      const data = rows.slice(1, 1 + CONFIG.csvPreviewRows);

      const table = h('table', { class: 'preview-table' }, [
        h('thead', {}, [h('tr', {}, head.map((c) => h('th', {}, [c])))]),
        h('tbody', {}, data.map((r) => h('tr', {},
          r.map((c) => h('td', {}, [c]))
        ))),
      ]);
      previewWrap.innerHTML = '';
      previewWrap.append(
        h('div', { class: 'muted', style: 'margin: 12px 0 6px' }, [
          `Разделитель: «${delimiter === '\t' ? '⇥ Tab' : delimiter}» · ` +
          `Колонок: ${head.length} · ` +
          `Строк данных: ${rows.length - 1}`,
        ]),
        table,
      );
    } catch (err) {
      previewWrap.innerHTML = '';
      toast('Не удалось прочитать файл: ' + err.message, 'error');
    }
  }

  async function doSend() {
    if (!chosenFile) return;
    sendBtn.disabled = true;
    progress.hidden = false;
    log.hidden = true;
    const bar = progress.querySelector('#progressBar');
    bar.style.width = '0%';

    try {
      const res = await uploadCsv(chosenFile, (p) => { bar.style.width = p + '%'; });
      log.hidden = false;

      const ok = res?.ok !== false;
      const head = ok
        ? `[OK] Импортировано: ${res?.imported ?? '?'}, ошибок: ${res?.failed ?? 0}`
        : `[FAIL] ${res?.error || 'неизвестная ошибка'}`;
      const warns = (res?.warnings && res.warnings.length)
        ? '\nПредупреждения:\n  - ' + res.warnings.join('\n  - ')
        : '';
      const tail = res?.log ? '\n---\n' + res.log : '';
      log.textContent = head + warns + tail;

      toast(ok ? 'CSV принят 1С' : 'Ошибка импорта', ok ? 'success' : 'error');
    } catch (err) {
      log.hidden = false;
      log.textContent = '[FAIL] ' + err.message;
      toast(err.message, 'error', 5000);
    } finally {
      sendBtn.disabled = !can('import');
    }
  }
}
