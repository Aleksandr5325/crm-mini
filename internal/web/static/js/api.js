// Обёртка над fetch для API бэкенда: JSON туда и обратно, ошибки — в ApiError.

export class ApiError extends Error {
  constructor(status, message, requestId) {
    super(message);
    this.status = status;       // HTTP-код; 0 — сервер недоступен
    this.requestId = requestId; // X-Request-ID: по нему запрос находится в логах сервера
  }
}

async function request(method, path, body, { raw = false } = {}) {
  const options = { method, headers: {} };

  if (body !== undefined) {
    if (raw) {
      options.body = body;
    } else {
      options.headers['Content-Type'] = 'application/json';
      options.body = JSON.stringify(body);
    }
  }

  let resp;
  try {
    resp = await fetch(`/api/v1${path}`, options);
  } catch {
    throw new ApiError(0, 'Сервер недоступен', null);
  }

  if (resp.status === 204) return null;

  const data = await resp.json().catch(() => null);
  if (!resp.ok) {
    throw new ApiError(resp.status, data?.error ?? `HTTP ${resp.status}`, resp.headers.get('X-Request-ID'));
  }

  return data;
}

// query собирает строку параметров, пропуская пустые значения.
function query(params) {
  const q = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') q.set(key, value);
  }
  const s = q.toString();
  return s ? `?${s}` : '';
}

export const api = {
  health: () => request('GET', '/health'),

  sources: () => request('GET', '/sources'),
  createSource: (source) => request('POST', '/sources', source),

  employees: () => request('GET', '/employees'),

  leads: (params = {}) => request('GET', `/leads${query(params)}`),
  lead: (id) => request('GET', `/leads/${id}`),
  createLead: (data) => request('POST', '/leads', data),
  updateLead: (id, patch) => request('PATCH', `/leads/${id}`, patch),
  deleteLead: (id) => request('DELETE', `/leads/${id}`),
  moveStage: (id, version, stage) => request('POST', `/leads/${id}/stage`, { version, stage }),
  stages: (id) => request('GET', `/leads/${id}/stages`),
  importLeads: (file) => request('POST', '/leads/import', file, { raw: true }),

  funnel: () => request('GET', '/reports/funnel'),
};
