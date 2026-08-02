const CACHE_NAME = 'nesio-shell-v1'
const API_CACHE_NAME = 'nesio-api-v1'
const APP_SHELL = ['/', '/index.html']

self.addEventListener('install', (event) => {
  event.waitUntil(caches.open(CACHE_NAME).then((cache) => cache.addAll(APP_SHELL)))
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim())
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

  if (url.origin === self.location.origin) {
    event.respondWith(cacheFirst(request, CACHE_NAME))
  }
})

async function cacheFirst(request, cacheName) {
  const cached = await caches.match(request)
  if (cached) {
    return cached
  }
  const response = await fetch(request)
  const cache = await caches.open(cacheName)
  cache.put(request, response.clone())
  return response
}

async function networkFirst(request, cacheName) {
  try {
    const response = await fetch(request)
    const cache = await caches.open(cacheName)
    cache.put(request, response.clone())
    return response
  } catch (error) {
    const cached = await caches.match(request)
    if (cached) {
      return cached
    }
    throw error
  }
}
