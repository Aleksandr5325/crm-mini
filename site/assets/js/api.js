// ============================================================================
//  Слой работы с API
// ============================================================================
//  Архитектура (см. config.js):
//    [фронт] ── multipart/form-data CSV ──▶ [1С HTTP-сервис]  (BASE_URL_1C)
//                                                  │
//                                                  ▼
//                                          [Go-бэк :8080]    (BASE_URL_BACK)
//                                                  │
//    [фронт] ◀── GET /leads, /sources, /leads/{id}/stages ─┘
// ============================================================================

import { CONFIG, getStagesOrdered } from './config.js';

// ---------------------------------------------------------------------------
//  Кэш мок-данных
// ---------------------------------------------------------------------------
let _mockLeads = null;
let _mockSources = null;
let _mockStagesHistory = {}; // { leadId: [{stage, entered_at, left_at}] }

async function loadMockLeads() {
  if (_mockLeads) return _mockLeads;
  // Чтобы между перезагрузками страницы сохранялись правки, синхронизируемся
  // с localStorage (как в первой версии CRM, эмулирующей 1С).
  const stored = localStorage.getItem('mockLeads');
  if (stored) {
    _mockLeads = JSON.parse(stored);
  } else {
    const r = await fetch(CONFIG.mockPaths.leads);
    if (!r.ok) throw new Error('Не удалось загрузить mock leads.json');
    _mockLeads = await r.json();
    localStorage.setItem('mockLeads', JSON.stringify(_mockLeads));
  }
  return _mockLeads;
}
async function loadMockSources() {
  if (_mockSources) return _mockSources;
  const r = await fetch(CONFIG.mockPaths.sources);
  if (!r.ok) throw new Error('Не удалось загрузить mock sources.json');
  _mockSources = await r.json();
  return _mockSources;
}
function persistMock() {
  if (_mockLeads) localStorage.setItem('mockLeads', JSON.stringify(_mockLeads));
}

// ---------------------------------------------------------------------------
//  Низкоуровневые fetch'и
// ---------------------------------------------------------------------------
function build1CHeaders(extra = {}) {
  const headers = { ...extra };
  if (CONFIG.auth.enabled) {
    const token = btoa(`${CONFIG.auth.user}:${CONFIG.auth.password}`);
    headers['Authorization'] = `Basic ${token}`;
  }
  return headers;
}

async function requestBack(path, options = {}) {
  const url = `${CONFIG.BASE_URL_BACK}${path}`;
  const resp = await fetch(url, {
    ...options,
    headers: options.headers || {},
  });
  if (!resp.ok) {
    const text = await resp.text().catch(() => '');
    throw new Error(`Back ${resp.status} ${resp.statusText}: ${text || '<пусто>'}`);
  }
  if (resp.status === 204) return null;
  const ct = resp.headers.get('content-type') || '';
  return ct.includes('application/json') ? resp.json() : resp.text();
}

async function request1C(method, path, payload) {
  const url = `${CONFIG.BASE_URL_1C}${path}`;
  const headers = {
    'Content-Type': 'application/json',
    'X-Target-Url': url,
    'token': 'skip-auth-token' // Токен для 1С
  };

  try {
    const resp = await fetch(`${CONFIG.BASE_URL_BACK}/proxy/1c`, {
      method: method,
      headers: headers,
      body: JSON.stringify(payload)
    });
    
    if (!resp.ok) {
      const text = await resp.text();
      console.warn(`[1C Sync Error] ${resp.status}:`, text);
    }
  } catch (err) {
    console.error('[1C Sync Network Error]', err);
  }
}

// ===========================================================================
//  ПУБЛИЧНЫЕ МЕТОДЫ
// ===========================================================================

/**
 * Загрузить CSV-файл в 1С (multipart/form-data).
 * 1С сама парсит и шлёт POST /import/leads JSON'ом в Go-бэк.
 *
 * Ожидаемый ответ 1С (любой, но рекомендуем):
 *   { ok: true, imported: 42, failed: 0, warnings: [], log: "..." }
 */
export async function uploadCsv(file, onProgress) {
  if (CONFIG.apiMode === 'mock') {
    if (onProgress) {
      for (let i = 0; i <= 100; i += 10) {
        onProgress(i);
        await new Promise((r) => setTimeout(r, 60));
      }
    }
    const text = await file.text();
    const rows = text.split(/\r?\n/).filter(Boolean);
    const imported = Math.max(0, rows.length - 1);
    return {
      ok: true,
      imported,
      failed: 0,
      warnings: [],
      log: `[MOCK] Файл "${file.name}" (${file.size} байт), строк данных: ${imported}.\n` +
           `[MOCK] В real-режиме файл уйдёт POST'ом в 1С на ${CONFIG.BASE_URL_1C}/import.`,
    };
  }

  // REAL: multipart → 1С
  //const form = new FormData();
  //form.append('file', file, file.name);
//
  //return new Promise((resolve, reject) => {
  //  const xhr = new XMLHttpRequest();
  //  xhr.open('POST', `${CONFIG.BASE_URL_1C}/import`);
  //  if (CONFIG.auth.enabled) {
  //    xhr.setRequestHeader(
  //      'Authorization',
  //      `Basic ${btoa(`${CONFIG.auth.user}:${CONFIG.auth.password}`)}`
  //    );
  //  }
  //  xhr.upload.onprogress = (e) => {
  //    if (e.lengthComputable && onProgress) {
  //      onProgress(Math.round((e.loaded / e.total) * 100));
  //    }
  //  };
  //  xhr.onload = () => {
  //    if (xhr.status >= 200 && xhr.status < 300) {
  //      try { resolve(JSON.parse(xhr.responseText)); }
  //      catch { resolve({ ok: true, raw: xhr.responseText }); }
  //    } else {
  //      reject(new Error(`1C ${xhr.status} ${xhr.statusText}: ${xhr.responseText}`));
  //    }
  //  };
  //  xhr.onerror = () => reject(new Error('Сеть недоступна или CORS заблокирован'));
  //  xhr.send(form);
  //});

  // REAL: отправка чистого текста (raw text) в 1С вместо FormData
  return new Promise((resolve, reject) => {
    file.text().then((text) => {
      const xhr = new XMLHttpRequest();
      xhr.open('POST', `${CONFIG.BASE_URL_1C}/import`);
      xhr.setRequestHeader('Content-Type', 'text/plain; charset=utf-8');
      
      // Авторизация (Basic, если включена)
      //if (CONFIG.auth.enabled) {
      //  xhr.setRequestHeader('Authorization', `Basic ${btoa(`${CONFIG.auth.user}:${CONFIG.auth.password}`)}`);
      //}

      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable && onProgress) {
          onProgress(Math.round((e.loaded / e.total) * 100));
        }
      };

      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try { resolve(JSON.parse(xhr.responseText)); }
          catch { resolve({ ok: true, raw: xhr.responseText }); }
        } else {
          reject(new Error(`1C ${xhr.status} ${xhr.statusText}: ${xhr.responseText}`));
        }
      };

      xhr.onerror = () => reject(new Error('Сеть недоступна или CORS заблокирован'));
      xhr.send(text);
    }).catch((err) => {
      reject(new Error('Не удалось прочитать файл: ' + err.message));
    });
  });
}

/**
 * Получить список лидов.
 * Реальный бэк (Go, см. lead.go::GetLeads) отдаёт ПЛОСКИЙ массив:
 *   [
 *     {
 *       lead_id: "LEAD-001",
 *       created_at: "...",
 *       source: "site",
 *       current_stage: "new",
 *       owner: "Петров П.П.",
 *       // расширенные поля (если бэкендер их добавит):
 *       full_name?, phone?, email?, comment?
 *     }, ...
 *   ]
 * Фронт оборачивает в { items, total } для совместимости с UI.
 *
 * @param {{source?: string, owner?: string}} params
 */
export async function getLeads(params = {}) {
  if (CONFIG.apiMode === 'mock') {
    const all = await loadMockLeads();
    return { items: all, total: all.length };
  }
  const qs = new URLSearchParams(
    Object.entries(params).filter(([, v]) => v !== undefined && v !== '')
  ).toString();
  const path = qs ? `/leads?${qs}` : '/leads';
  const arr = await requestBack(path);
  const items = Array.isArray(arr) ? arr : [];
  return { items, total: items.length };
}

/**
 * Получить одну сделку по id.
 * Реальный бэк такого метода НЕ предоставляет, поэтому ищем в общем списке.
 */
export async function getLead(id) {
  if (CONFIG.apiMode === 'mock') {
    const all = await loadMockLeads();
    const item = all.find((x) => String(x.lead_id) === String(id));
    if (!item) throw new Error(`Лид ${id} не найден`);
    return item;
  }
  // На бэке нет /leads/{id} — приходится тянуть весь список и фильтровать.
  // (Когда бэкендер добавит GET /leads/{id}, заменить на requestBack(`/leads/${id}`).)
  const arr = await requestBack('/leads');
  const item = (arr || []).find((x) => String(x.lead_id) === String(id));
  if (!item) throw new Error(`Лид ${id} не найден на сервере`);
  return item;
}

/**
 * Создать нового лида.
 * Реальный бэк: POST /leads (см. main.go)
 */
/**
 * Создать нового лида.
 * Реальный бэк: POST /leads (см. main.go)
 * Плюс дублирование создания лида в 1С (POST).
 */
export async function createLead(leadData) {
  if (CONFIG.apiMode === 'mock') {
    if (!_mockLeads) await loadMockLeads();
    let maxId = 0;
    _mockLeads.forEach(lead => {
      const num = parseInt(lead.lead_id, 10);
      if (!isNaN(num) && num > maxId) maxId = num;
    });
    const newId = maxId + 1; // Числовой ID, как и требовалось
    const newLead = {
      lead_id: newId,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      current_stage: 'new',
      ...leadData
    };
    _mockLeads.push(newLead);
    localStorage.setItem('mockLeads', JSON.stringify(_mockLeads));
    return { ok: true, lead: newLead };
  }

  // 1. Создаем лид в Go (PostgreSQL)
  const lead = await requestBack('/leads', {
    method: 'POST',
    body: JSON.stringify(leadData),
  });

  // 2. Отправляем в 1С через прокси (конвертируем ID в число)
  await request1C('POST', '/leads', {
    lead_id: parseInt(lead.lead_id, 10), 
    source: lead.source,
    current_stage: lead.current_stage,
    manager: lead.owner, // Согласно твоей OpenAPI
    created_at: lead.created_at.replace('Z', ''),
    note: lead.comment || '',
    full_name: lead.full_name || '',
    email: lead.email || '',
    token: 'skip-auth-token' // <--- ДОБАВЬ ЭТУ СТРОКУ СЮДА
  });

  return lead;
}

/**
 * Обновить сделку. Бэк такой ручки пока не предоставляет.
 * В mock-режиме правим in-memory + localStorage; в real — кидаем понятную ошибку.
 */
export async function updateLead(id, patch) {
  if (CONFIG.apiMode === 'mock') {
    const all = await loadMockLeads();
    const idx = all.findIndex((x) => String(x.lead_id) === String(id));
    if (idx < 0) throw new Error(`Лид ${id} не найден`);
    all[idx] = { ...all[idx], ...patch, updated_at: new Date().toISOString() };
    persistMock();
    return all[idx];
  }
  
  // 1. Обновляем в Go
  const lead = await requestBack(`/leads/${encodeURIComponent(id)}`, {
    method: 'PUT', // <--- Заменили PATCH на PUT
    headers: {
      'Content-Type': 'application/json' 
    },
    body: JSON.stringify(patch),
  });

  // 2. Собираем только измененные поля по спецификации OpenAPI
  const payload1c = {
    token: 'skip-auth-token' // <--- ДОБАВЬ ЭТУ СТРОКУ СЮДА
  };
  if (patch.full_name !== undefined) payload1c.full_name = patch.full_name;
  if (patch.email !== undefined) payload1c.email = patch.email;
  if (patch.source !== undefined) payload1c.source = patch.source;
  if (patch.owner !== undefined) payload1c.manager = patch.owner;
  if (patch.comment !== undefined) payload1c.note = patch.comment;

  if (Object.keys(payload1c).length > 0) {
    // 1С ожидает обновление по URL: /leads/{id}
    await request1C('PUT', `/leads/${id}`, payload1c);
  }

  return lead;
}
/**
 * Перевести лид на другую стадию.
 * Реальный бэк: POST /leads/{id}/move (см. main.go)
 */
/**
 * Перевести лид на другую стадию.
 * Реальный бэк: POST /leads/{id}/move (см. main.go)
 * Плюс дублирование обновления стадии в 1С (PATCH).
 */
export async function moveLead(leadId, targetStageCode) {
  if (CONFIG.apiMode === 'mock') {
    if (!_mockLeads) await loadMockLeads();
    const lead = _mockLeads.find((x) => String(x.lead_id) === String(leadId));
    if (lead) {
      lead.current_stage = targetStageCode;
      lead.updated_at = new Date().toISOString();
      localStorage.setItem('mockLeads', JSON.stringify(_mockLeads));
    }
    if (!_mockStagesHistory[leadId]) _mockStagesHistory[leadId] = [];
    _mockStagesHistory[leadId].push({
      lead_id: leadId,
      stage: targetStageCode,
      entered_at: new Date().toISOString(),
      left_at: null
    });
    return { ok: true };
  }

  // 1. Перемещаем на канбан-доске (Go)
  const lead = await requestBack(`/leads/${encodeURIComponent(leadId)}/move`, {
    method: 'POST',
    body: JSON.stringify({ stage: targetStageCode }),
  });

  // 2. Синхронизируем новую стадию с 1С
  await request1C('PUT', `/leads/${leadId}`, {
    current_stage: targetStageCode
  });

  return { ok: true, lead };
}

/**
 * Список стадий воронки. Источник правды — config.STAGES (зашит enum'ом
 * из validateStages в Go-бэке), фронт всегда работает с ним.
 */
export async function getStages() {
  return getStagesOrdered();
}

/**
 * Получить историю стадий конкретного лида.
 * Реальный бэк: GET /leads/{id}/stages (см. lead.go::GetLeadStages).
 *   [{ lead_id, stage, entered_at, left_at }, ...]
 */
export async function getLeadStages(leadId) {
  if (CONFIG.apiMode === 'mock') {
    return _mockStagesHistory[leadId] || [];
  }
  const arr = await requestBack(`/leads/${encodeURIComponent(leadId)}/stages`);
  return Array.isArray(arr) ? arr : [];
}

/**
 * Список источников.
 * Реальный бэк: GET /sources → [{code, name}, ...] (см. source.go)
 */
export async function getSources() {
  if (CONFIG.apiMode === 'mock') {
    return loadMockSources();
  }
  const arr = await requestBack('/sources');
  return Array.isArray(arr) ? arr : [];
}

/**
 * Справочники для фильтров: источники + менеджеры (owners).
 * Поскольку бэк отдельной ручки для owners не имеет, собираем их из /leads.
 */
export async function getDictionaries() {
  const [sources, leadsResp] = await Promise.all([getSources(), getLeads()]);
  const owners = [...new Set((leadsResp.items || [])
    .map((x) => x.owner || x.manager_name)
    .filter(Boolean))];
  return { sources, owners };
}

// Экспорт для отладки из консоли (window.__api)
if (typeof window !== 'undefined') {
  window.__api = {
    uploadCsv, getLeads, getLead, createLead, updateLead, moveLead,
    getStages, getLeadStages, getSources, getDictionaries,
    config: CONFIG,
  };
}
