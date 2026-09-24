// ============================================================================
//  Конфигурация фронтенда
// ============================================================================
//  Архитектура цепочки:
//
//    [фронт] ─── multipart/form-data CSV ───▶ [1С HTTP-сервис]
//                                                │
//                                                │ POST /import/leads (JSON)
//                                                ▼
//                                            [бэк Go :8080] ──▶ PostgreSQL
//                                                │
//    [фронт] ◀── GET /leads, /sources, ... ──┘
//
//  Поэтому у фронта ДВА базовых URL:
//    - BASE_URL_1C   — для загрузки CSV  (импорт)
//    - BASE_URL_BACK — для чтения данных (лиды, источники, история)
//
//  Переключение режима mock/real:
//    localStorage.setItem('apiMode', 'real')
// ============================================================================

const STORED_MODE = (typeof localStorage !== 'undefined')
  ? localStorage.getItem('apiMode')
  : null;

export const CONFIG = {
  // 'mock' | 'real'
  apiMode: STORED_MODE || 'real',

  // URL HTTP-сервиса 1С (куда фронт льёт CSV при импорте).
  BASE_URL_1C: 'http://localhost/CRM_mini/hs/crm',

  // URL Go-бэкенда (см. main.go: ListenAndServe(":8080")).
  BASE_URL_BACK: 'http://localhost:8080',

  // Если 1С требует Basic-аутентификацию — заполни.
  auth: {
    enabled: true,

    // Basic auth для 1С
    user: 'admin',
    password: 'admin',

    // token из документации API 1С
    token: 'my-secret-token'
  },

  // Зафиксированный enum стадий из бэка (см. validateStages в import_json.go).
  // ПОРЯДОК ВАЖЕН — он же используется в правиле "нельзя двигать назад".
  STAGES: [
    { code: 'new',        title: 'Новый',                    order: 1, kind: 'open'  },
    { code: 'qualified',  title: 'Квалифицирован',           order: 2, kind: 'open'  },
    { code: 'proposal',   title: 'Коммерческое предложение', order: 3, kind: 'open'  },
    { code: 'won',        title: 'Закрыт: выигран',          order: 4, kind: 'won'   },
    { code: 'lost',       title: 'Закрыт: проигран',         order: 4, kind: 'lost'  },
  ],

  // Пути к мок-файлам (относительно index.html)
  mockPaths: {
    leads:   './mock/leads.json',
    sources: './mock/sources.json',
  },

  // Сколько строк CSV показывать в превью на экране импорта
  csvPreviewRows: 10,

  // Пагинация в списке лидов
  pageSize: 20,
};

// Удобный хелпер для смены режима из UI / DevTools
export function setApiMode(mode) {
  if (!['mock', 'real'].includes(mode)) {
    throw new Error(`Unknown apiMode: ${mode}`);
  }
  CONFIG.apiMode = mode;
  localStorage.setItem('apiMode', mode);
}

// ----------------------------------------------------------------------------
//  Хелперы воронки
// ----------------------------------------------------------------------------

/** Стадии в правильном порядке (отсортированы по order) */
export function getStagesOrdered() {
  return [...CONFIG.STAGES].sort((a, b) => a.order - b.order);
}

/** Найти стадию по коду */
export function getStageByCode(code) {
  return CONFIG.STAGES.find((s) => s.code === code) || null;
}

/**
 * Можно ли переместить лид со стадии `from` на стадию `to`.
 * Правила:
 *   1. Из won/lost — никуда (это конечные стадии).
 *   2. На стадию, у которой order меньше текущей — нельзя (запрет движения назад).
 *   3. На ту же самую стадию — нет смысла.
 */
export function canMoveStage(from, to) {
  if (from === to) return false;
  const f = getStageByCode(from);
  const t = getStageByCode(to);
  if (!f || !t) return false;
  if (f.kind === 'won' || f.kind === 'lost') return false;
  return t.order > f.order;
}
