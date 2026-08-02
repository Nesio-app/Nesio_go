import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' }
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
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
  get: (localDay?: string, slot?: string, minSeverity?: string) =>
    api.get('/today', { params: { local_day: localDay, slot, min_severity: minSeverity } }),
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
  auth: (provider: string, credentials: any) =>
    api.post(`/connectors/${provider}/auth`, { credentials }),
  delete: (id: string) =>
    api.delete(`/connectors/${id}`),
  sync: (id: string) =>
    api.post(`/connectors/${id}/sync`),
}

export const memories = {
  list: () => api.get('/memories'),
  create: (data: any) => api.post('/memories', data),
}

export const signals = {
  create: (data: any) => api.post('/signals', data),
}

export const user = {
  me: () => api.get('/me'),
  update: (data: any) => api.patch('/me', data),
}
