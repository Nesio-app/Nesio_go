import axios, { AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { enqueueWriteRequest } from './offlineQueue'

const apiOrigin = (import.meta.env.VITE_API_URL ?? '').replace(/\/$/, '')

const api = axios.create({
  baseURL: `${apiOrigin}/api/v1`,
  headers: { 'Content-Type': 'application/json' }
})

api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }

  const method = (config.method ?? 'get').toLowerCase()
  if (!navigator.onLine && ['post', 'patch', 'put', 'delete'].includes(method)) {
    enqueueWriteRequest({
      url: `${config.baseURL ?? ''}${config.url ?? ''}`,
      method: method.toUpperCase(),
      body: typeof config.data === 'string' ? config.data : JSON.stringify(config.data ?? {}),
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    })
    return Promise.reject({ queued: true, message: 'Request queued for sync when connection returns.' })
  }

  return config
})

api.interceptors.response.use(
  (res: AxiosResponse) => res,
  (err: { response?: { status?: number } }) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      window.dispatchEvent(new Event('nesio:unauthorized'))
    }
    return Promise.reject(err)
  }
)

export default api

export const auth = {
  login: (email: string, password: string) =>
    api.post('/auth/login', { email, password }),
  register: (email: string, password: string) =>
    api.post('/auth/register', { email, password }),
  forgotPassword: (email: string) =>
    api.post<{ reset_token: string; expires_at: string }>('/auth/forgot-password', { email }),
  resetPassword: (email: string, token: string, password: string) =>
    api.post('/auth/reset-password', { email, token, password }),
}

export const today = {
  get: (localDay?: string) =>
    api.get('/today', { params: { local_day: localDay } }),
  dismiss: (id: string) =>
    api.post(`/cards/${id}/dismiss`),
  mute: (id: string) =>
    api.post(`/cards/${id}/mute`),
  done: (id: string) =>
    api.post(`/cards/${id}/done`),
}

export const tasks = {
  list: (status?: string) =>
    api.get('/tasks', { params: { status } }),
  create: (data: any) =>
    api.post('/tasks', data),
  update: (id: string, data: any) =>
    api.patch(`/tasks/${id}`, data),
}

export const chat = {
  send: (message: string, tier?: string) =>
    api.post('/chat', { message, tier }),
  history: () =>
    api.get('/chat/history'),
}

export const ask = {
  query: (question: string) =>
    api.post<{ type: string; answer: string; sources?: Array<Record<string, any>> }>('/ask', { question }),
}

export const connectors = {
  list: () => api.get('/connectors'),
  providers: () => api.get('/connectors/providers'),
  auth: (provider: string) =>
    api.post(`/connectors/${provider}/auth`),
  import: (provider: string, payload: Record<string, any>, sync = true) =>
    api.post(`/connectors/${provider}/import`, payload, { params: { sync: sync ? '1' : '0' } }),
  delete: (id: string) =>
    api.delete(`/connectors/${id}`),
  sync: (id: string) =>
    api.post(`/connectors/${id}/sync`),
}

export const gmail = {
  inbox: (params?: { box?: 'inbox' | 'sent'; q?: string }) => api.get('/connectors/gmail/inbox', { params }),
  send: (data: { to: string; subject: string; body: string }) =>
    api.post('/connectors/gmail/send', data),
  authorizeUrl: () => api.get('/connectors/gmail/oauth/authorize'),
}

export const memories = {
  list: (params?: { domain?: string }) => api.get('/memories', { params }),
  create: (data: { title: string; body?: string; tags?: string[] }) =>
    api.post('/memories', data),
}

export const domains = {
  overview: () => api.get('/domains/overview'),
  detail: (domain: string) => api.get(`/domains/${encodeURIComponent(domain)}/detail`),
  createTask: (domain: string, data: { title: string; due_date?: string | null; tags?: string[] }) =>
    api.post(`/domains/${encodeURIComponent(domain)}/tasks`, data),
  createMemory: (domain: string, data: { title: string; body?: string; tags?: string[] }) =>
    api.post(`/domains/${encodeURIComponent(domain)}/memories`, data),
  updateNode: (domain: string, id: string, data: { title?: string; body?: string; status?: string; due_date?: string | null }) =>
    api.patch(`/domains/${encodeURIComponent(domain)}/nodes/${id}`, data),
  deleteNode: (domain: string, id: string) => api.delete(`/domains/${encodeURIComponent(domain)}/nodes/${id}`),
}

export const rooms = {
  list: () => api.get('/rooms'),
  create: (data: { name: string; icon?: string; sort_order?: number }) => api.post('/rooms', data),
  update: (id: string, data: { name?: string; icon?: string; sort_order?: number }) => api.patch(`/rooms/${id}`, data),
  remove: (id: string) => api.delete(`/rooms/${id}`),
}

export const containers = {
  list: (roomId?: string) => api.get('/containers', { params: { room: roomId } }),
  create: (data: { name: string; icon?: string; room_id?: string | null; sort_order?: number }) => api.post('/containers', data),
  update: (id: string, data: { name?: string; icon?: string; room_id?: string | null; sort_order?: number }) => api.patch(`/containers/${id}`, data),
  remove: (id: string) => api.delete(`/containers/${id}`),
}

export const items = {
  list: (params?: { room?: string; container?: string; q?: string }) => api.get('/items', { params }),
  get: (id: string) => api.get(`/items/${id}`),
  create: (data: {
    name: string
    body?: string
    room_id?: string | null
    container_id?: string | null
    location_note?: string
    expiry_date?: string
    is_document?: boolean
    document_type?: string
    document_number?: string
    quantity?: number
    unit?: string
    primary_image_url?: string
    visual_hash?: string
    reminder_label?: string
    tags?: string[]
  }) => api.post('/items/create', data),
  update: (id: string, data: any) => api.patch(`/items/${id}`, data),
  remove: (id: string) => api.delete(`/items/${id}`),
  whereIs: (q: string) => api.get('/items/where-is', { params: { q } }),
  whereIsPhoto: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return api.post('/items/where-is-photo', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
  },
  expiring: () => api.get('/items/expiring'),
  documents: () => api.get('/items/documents'),
  duplicate: (id: string, targetItemId: string, increment = 1) =>
    api.post(`/items/${id}/duplicate`, { target_item_id: targetItemId, increment }),
  snoozeExpiry: (id: string) => api.post(`/items/${id}/snooze-expiry`),
  analyze: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return api.post('/vision/analyze', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
  },
}

export const vision = {
  analyze: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return api.post('/vision/analyze', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
  },
}

export const reminders = {
  list: () => api.get('/reminders'),
  create: (data: { title: string; remind_at?: string; source?: string; body?: string; important?: boolean; node_id?: string }) =>
    api.post('/reminders', data),
  done: (id: string) => api.post(`/reminders/${id}/done`),
}

export const medications = {
  list: () => api.get('/medications'),
  create: (data: { name: string; dosage?: string; frequency?: string; start_date?: string; end_date?: string; image_url?: string }) =>
    api.post('/medications', data),
}

export const medicine = {
  list: () => api.get('/medicine'),
  ocr: (data: { name: string; dosage?: string; frequency?: string; start_date?: string; end_date?: string; image_url?: string }) =>
    api.post('/medicine/ocr', data),
  reminder: (id: string) => api.post(`/medicine/${id}/reminder`),
}

export const dailyBriefs = {
  today: () => api.get('/daily-briefs/today'),
  generate: () => api.post('/daily-briefs/generate'),
  byDay: (day: string) => api.get('/daily-brief', { params: { day } }),
  read: (id: string) => api.post(`/daily-brief/${id}/read`),
}

export const intake = {
  ingest: (text: string) => api.post<{
    node_id: string
    reminder_created: boolean
    intent: string
    intent_label: string
    confidence: number
    remind_at?: string | null
  }>('/intake/ingest', { text }),
  upload: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return api.post('/intake/upload', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
  },
}

export const search = {
  query: (q: string, limit?: number) => api.get('/search', { params: { q, limit } }),
}

export const mention = {
  query: (q: string, limit?: number) => api.get('/nodes/mention', { params: { q, limit } }),
}

export const extraction = {
  analyze: (text: string) => api.post<{ extracted: Array<Record<string, any>> }>('/extraction/analyze', { text }),
  upload: (files: File[], note?: string) => {
    const formData = new FormData()
    files.forEach((file) => formData.append('files', file))
    if (note) {
      formData.append('note', note)
    }
    return api.post<{ extracted: Array<Record<string, any>> }>('/extraction/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
}

export const relations = {
  list: (nodeId?: string) => api.get('/relations', { params: { node_id: nodeId } }),
  create: (data: { from_node: string; to_node: string; relation: string }) => api.post('/relations', data),
  remove: (id: string) => api.delete(`/relations/${id}`),
}

export const user = {
  me: () => api.get('/me'),
  update: (data: any) => api.patch('/me', data),
}

export const dataExport = {
  run: () => api.post('/export'),
}
