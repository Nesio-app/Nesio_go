import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import App from './App'
import { flushQueuedRequests, setupOfflineQueueSync } from './api/offlineQueue'
import './styles/index.css'

const queryClient = new QueryClient()

if ('serviceWorker' in navigator) {
  void navigator.serviceWorker.register('/sw.js')
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
