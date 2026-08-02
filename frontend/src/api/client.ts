import axios, { AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { enqueueWriteRequest } from './offlineQueue'

const api = axios.create({
  baseURL: '/api/v1',
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
      window.location.href = '/login'
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

export const connectors = {
  list: () => api.get('/connectors'),
  auth: (provider: string) =>
    api.post(`/connectors/${provider}/auth`),
  delete: (id: string) =>
    api.delete(`/connectors/${id}`),
  sync: (id: string) =>
    api.post(`/connectors/${id}/sync`),
}

export const gmail = {
  inbox: () => api.get('/connectors/gmail/inbox'),
  send: (data: { to: string; subject: string; body: string }) =>
    api.post('/connectors/gmail/send', data),
}

export const memories = {
  list: () => api.get('/memories'),
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

export const user = {
  me: () => api.get('/me'),
  update: (data: any) => api.patch('/me', data),
}
