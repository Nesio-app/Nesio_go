const CACHE_NAME = 'nesio-shell-v3'
const API_CACHE_NAME = 'nesio-api-v3'
const APP_BASE = new URL(self.registration.scope).pathname
const APP_SHELL = [APP_BASE, `${APP_BASE}index.html`]

self.addEventListener('install', (event) => {
  event.waitUntil(caches.open(CACHE_NAME).then((cache) => cache.addAll(APP_SHELL)))
  self.skipWaiting()
})

self.addEventListener('message', (event) => {
  if (event.data?.type === 'SKIP_WAITING') {
    self.skipWaiting()
  }
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(
        keys
          .filter((key) => key !== CACHE_NAME && key !== API_CACHE_NAME)
          .map((key) => caches.delete(key)),
      ))
      .then(() => self.clients.claim()),
  )
})

self.addEventListener('fetch', (event) => {
  const request = event.request
  if (request.method !== 'GET') {
    return
  }

  const url = new URL(request.url)
  const isApiGet = url.pathname.startsWith('/api/v1/')

  if (isApiGet) {
    event.respondWith(networkFirst(request, API_CACHE_NAME))
    return
  }

  if (request.mode === 'navigate') {
    event.respondWith(networkFirst(request, CACHE_NAME))
    return
  }

  if (url.origin === self.location.origin) {
    if (request.destination === 'script' || request.destination === 'style') {
      event.respondWith(networkFirst(request, CACHE_NAME))
      return
    }

    event.respondWith(cacheFirst(request, CACHE_NAME))
  }
})

async function cacheFirst(request, cacheName) {
  const cached = await caches.match(request)
  if (cached) {
    return cached
  }
  const response = await fetch(request)
  if (response.ok) {
    const cache = await caches.open(cacheName)
    cache.put(request, response.clone())
  }
  return response
}

async function networkFirst(request, cacheName) {
  try {
    const response = await fetch(request)
    if (response.ok) {
      const cache = await caches.open(cacheName)
      cache.put(request, response.clone())
    }
    return response
  } catch (error) {
    const cached = await caches.match(request)
    if (cached) {
      return cached
    }
    throw error
  }
}
