# CRM — фронт для курсовой (1С + Go-бэк + PostgreSQL)

Чистый HTML/CSS/JS, ES-модули, без сборки. Готовый прототип фронта.

## Архитектура

```
   ┌──────────┐                          ┌──────────────────┐
   │  фронт   │ ── multipart CSV ──▶     │  1С HTTP-сервис  │
   │ (этот    │     (Импорт CSV)         │   BASE_URL_1C    │
   │  репо)   │                          └────────┬─────────┘
   │          │                                   │ POST /import/leads (JSON)
   │          │                                   ▼
   │          │                          ┌──────────────────┐
   │          │ ◀── GET /leads          │   Go-бэк :8080   │
   │          │ ◀── GET /sources         │   BASE_URL_BACK  │
   │          │ ◀── GET /leads/{id}/stages└────────┬─────────┘
   └──────────┘                                   │
                                                  ▼
                                          ┌──────────────────┐
                                          │   PostgreSQL     │
                                          └──────────────────┘
```

CSV льётся из фронта в 1С. 1С парсит и POST-ит JSON в Go-бэк. Фронт читает данные напрямую с Go-бэка через GET-эндпоинты.

## Что внутри

```
site_v3/
├─ index.html
├─ assets/
│  ├─ css/style.css
│  └─ js/
│     ├─ app.js              точка входа, hash-роутинг, обработка ролей
│     ├─ config.js           BASE_URL_1C, BASE_URL_BACK, enum стадий, правила движения
│     ├─ api.js              все методы (uploadCsv, getLeads, getLeadStages, ...)
│     ├─ auth.js             3 роли + матрица прав
│     ├─ utils.js            h(), parseCsv(), formatDate(), toast()
│     └─ views/
│        ├─ importView.js    экран импорта CSV
│        ├─ leadsView.js     таблица лидов с фильтрами/сортировкой
│        ├─ kanbanView.js    канбан + статистика + запрет обратного движения
│        └─ dealView.js      карточка лида + история стадий
├─ mock/
│  ├─ leads.json             10+ тестовых лидов в формате бэка
│  ├─ sources.json           5 источников
│  └─ example_import.csv     пример CSV для теста импорта
└─ README.md
```

## Как запустить

`index.html` использует ES-модули и `fetch` к JSON — открыть его двойным кликом нельзя, нужен локальный http-сервер.

```bash
cd site_v3
python3 -m http.server 5500
# открыть http://localhost:5500
```

Альтернативы: `npx serve -l 5500`, расширение **Live Server** в VSCode.

## Режим API: mock и real

По умолчанию — `mock`: фронт читает `mock/leads.json` и `mock/sources.json`, сохраняет правки в `localStorage`. Можно работать без бэка и без 1С.

Перейти на `real`:
```js
localStorage.setItem('apiMode', 'real'); location.reload();
```
Вернуться:
```js
localStorage.setItem('apiMode', 'mock'); location.reload();
```

Адреса 1С и бэка живут в `assets/js/config.js`:
```js
BASE_URL_1C:   'http://localhost/crm_base/hs/crm',
BASE_URL_BACK: 'http://localhost:8080',
```

## Роли

Кнопки в правом верхнем углу:

| Роль          | Просмотр | DnD по канбану | Создание | Редактирование | Импорт CSV |
|---------------|:--------:|:--------------:|:--------:|:--------------:|:----------:|
| Пользователь  |    ✔     |        —       |     —    |        —       |      —     |
| Работник      |    ✔     |        ✔       |     —    |        —       |      —     |
| Менеджер      |    ✔     |        ✔       |     ✔    |        ✔       |      ✔     |

Текущая роль хранится в `localStorage` (ключ `currentRole`).

## Правила движения по воронке

Захардкожены в `config.js → canMoveStage(from, to)`:

1. **Только вперёд.** `order` целевой стадии должен быть строго больше текущей.
2. **Из won/lost — никуда.** Это финальные стадии: «попал в закрытый — значит закрыт».
3. На ту же стадию переносить нельзя (нет смысла).

UI:
- В тулбаре подсказка про правила.
- Колонки won/lost имеют свой цвет.
- Карточки в won/lost помечены 🔒 и не draggable.
- При попытке drop на запрещённую колонку — красная подсветка + тост с причиной.

## Контракт API (что фронт ожидает от Go-бэка)

Базовый URL: `BASE_URL_BACK` (по умолчанию `http://localhost:8080`).

### `GET /leads?source=&owner=` — список лидов

Сейчас бэк возвращает (см. `lead.go::GetLeads`):
```json
[
  {
    "lead_id": "LEAD-001",
    "created_at": "2026-03-20T10:30:00Z",
    "source": "site",
    "current_stage": "new",
    "owner": "Петров П.П."
  }
]
```
Фронт оборачивает в `{ items, total }` сам.

### `GET /leads/{id}/stages` — история стадий

```json
[
  { "lead_id": "LEAD-001", "stage": "new",       "entered_at": "...", "left_at": "..."  },
  { "lead_id": "LEAD-001", "stage": "qualified", "entered_at": "...", "left_at": null   }
]
```

### `GET /sources` — справочник источников

```json
[
  { "code": "site",       "name": "Сайт компании" },
  { "code": "olymp",      "name": "Олимпиады/Конференции" },
  { "code": "aggregator", "name": "Агрегаторы/Партнёры" }
]
```

### `POST /import/leads` — приём JSON от 1С

Этот эндпоинт **фронт не вызывает напрямую** — его дёргает 1С после парсинга CSV. Сам формат см. `import_json.go`:
```json
{
  "import_id": "imp-2026-05-15-001",
  "sources":  [{"code": "site", "name": "Сайт компании"}],
  "leads": [
    {
      "lead_id": "LEAD-101",
      "created_at": "2026-05-15T10:00:00Z",
      "source": "site",
      "current_stage": "new",
      "owner": "Петров П.П.",
      "stages": [
        { "stage": "new", "entered_at": "2026-05-15T10:00:00Z", "left_at": null }
      ]
    }
  ]
}
```

### `POST` в 1С — приём CSV от фронта

Фронт льёт `multipart/form-data` с полем `file` на `BASE_URL_1C/import`. Ответ ожидаем в формате:
```json
{ "ok": true, "imported": 42, "failed": 0, "warnings": [], "log": "..." }
```
Поле `log` отображается на экране импорта как протокол обработки.

## Как добавить тестовые лиды

Четыре способа — от самого удобного:

1. **Кнопка «+ Новый лид»** в тулбаре экрана «Лиды» (требует роль «Менеджер»). Открывает форму, при сабмите создаёт лид в стадии `new` и записывает первую запись в историю стадий. В mock-режиме сохраняется в `localStorage`, в real — требует `POST /leads` на бэке.

2. **Импорт CSV** — основной сценарий курсовой. Пример: `mock/example_import.csv`. Менеджер заходит на экран «Импорт CSV», выбирает файл, жмёт «Отправить в 1С». В mock-режиме просто посчитаются строки; в real-режиме файл уйдёт в 1С, которая разберёт его и POST-нет JSON в `POST /import/leads`.

3. **Руками в `mock/leads.json`** — добавляешь объект в массив. Подвох: при первом запуске фронт копирует JSON в `localStorage` и потом работает оттуда. Чтобы перечитать свежий файл:
   ```js
   localStorage.removeItem('mockLeads'); location.reload();
   ```

4. **Из DevTools** через `window.__api`:
   ```js
   await __api.createLead({
     full_name: "Тестов Тест Тестович",
     phone: "+7...", email: "t@t.ru",
     source: "site", owner: "Петров П.П.",
   });
   location.reload();
   ```

## ⚠️ Что нужно добавить на бэк (для полноценной работы фронта)

Сейчас в `main.go` нет двух эндпоинтов. Без них **редактирование лида и DnD-смена стадии работают только в mock-режиме** (изменения остаются в `localStorage`):

### 0) `POST /leads` — создание лида

```http
POST /leads
Content-Type: application/json

{
  "full_name": "Иванов И.И.",
  "phone":     "+7...",
  "email":     "...",
  "source":    "site",
  "current_stage": "new",
  "owner":     "..."
}
```
Должен сгенерировать `lead_id`, поставить `created_at = now`, создать первую запись в `lead_stages` (stage=`new`, entered_at=`now`, left_at=`null`). Ответ — созданный `Lead`.

### 1) `PATCH /leads/{id}` — частичное обновление

```http
PATCH /leads/LEAD-001
Content-Type: application/json

{
  "full_name": "Иванов И.И.",
  "phone":     "+7...",
  "email":     "...",
  "source":    "site",
  "owner":     "...",
  "comment":   "..."
}
```
Ответ — обновлённый `Lead`.

### 2) `POST /leads/{id}/move` — смена стадии (для канбана)

```http
POST /leads/LEAD-001/move
Content-Type: application/json

{ "stage": "qualified" }
```
Должен на стороне Go валидировать `canMoveStage`-правила, чтобы фронт не был единственным защитником (а заодно и записать запись в `lead_stages` с `entered_at = now`, `left_at = null`, плюс закрыть прошлую `left_at = now`).

### 3) Расширение модели `Lead`

Бэк сейчас хранит только `LeadID`, `CreatedAt`, `Source`, `CurrentStage`, `Owner`. CSV/фронт работают также с **`FullName`, `Phone`, `Email`, `Comment`**. Добавь эти поля в `models.Lead` и в структуру `LeadImport` (`import_json.go`), иначе при импорте они потеряются.

### 4) CORS

Фронт открывается с `http://localhost:5500`, а бэк на `:8080`. Чтобы браузер не блокировал кросс-доменные запросы, в `main.go` нужно подключить мидлварь:

```go
r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin",  "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if req.Method == "OPTIONS" { w.WriteHeader(http.StatusOK); return }
        next.ServeHTTP(w, req)
    })
})
```

Или подключить готовый `github.com/rs/cors`.

## Чек-лист на защиту

- [x] Импорт CSV (превью + прогресс + лог результата).
- [x] Список лидов: поиск, фильтры (дата, стадия, источник, менеджер), сортировка, пагинация.
- [x] Канбан-воронка с DnD и **запретом возврата + блокировкой won/lost**.
- [x] Карточка лида с историей стадий (GET `/leads/{id}/stages`).
- [x] Три роли с разной матрицей прав.
- [x] Статистика над канбаном: всего, конверсия, среднее время.
- [x] Готовый слой API с режимами mock/real.
- [x] Документация по контракту и тому что добить на бэке.
