import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import App from './App'
import { flushQueuedRequests, setupOfflineQueueSync } from './api/offlineQueue'
import './styles/index.css'

const queryClient = new QueryClient()

if ('serviceWorker' in navigator) {
  void navigator.serviceWorker
    .register(`${import.meta.env.BASE_URL}sw.js`)
    .then((registration) => {
      const activateWaitingWorker = () => {
        if (registration.waiting) {
          registration.waiting.postMessage({ type: 'SKIP_WAITING' })
        }
      }

      registration.addEventListener('updatefound', () => {
        const installingWorker = registration.installing
        if (!installingWorker) {
          return
        }

        installingWorker.addEventListener('statechange', () => {
          if (installingWorker.state === 'installed' && navigator.serviceWorker.controller) {
            activateWaitingWorker()
          }
        })
      })

      void registration.update()
      activateWaitingWorker()
    })

  let reloadedForSwUpdate = false
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    if (reloadedForSwUpdate) {
      return
    }
    reloadedForSwUpdate = true
    window.location.reload()
  })
}
setupOfflineQueueSync()
void flushQueuedRequests()

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>,
)
