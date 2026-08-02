type OfflineQueueEntry = {
  url: string
  method: string
  body?: string
  headers: Record<string, string>
}

const QUEUE_KEY = 'nesio-offline-write-queue'

function readQueue(): OfflineQueueEntry[] {
  try {
    const raw = localStorage.getItem(QUEUE_KEY)
    return raw ? (JSON.parse(raw) as OfflineQueueEntry[]) : []
  } catch {
    return []
  }
}

function writeQueue(entries: OfflineQueueEntry[]) {
  localStorage.setItem(QUEUE_KEY, JSON.stringify(entries))
}

export function enqueueWriteRequest(entry: OfflineQueueEntry) {
  const entries = readQueue()
  entries.push(entry)
  writeQueue(entries)
}

export async function flushQueuedRequests() {
  const entries = readQueue()
  if (entries.length === 0 || !navigator.onLine) {
    return
  }

  const remaining: OfflineQueueEntry[] = []
  for (const entry of entries) {
    try {
      const response = await fetch(entry.url, {
        method: entry.method,
        headers: entry.headers,
        body: entry.body,
      })
      if (!response.ok) {
        remaining.push(entry)
      }
    } catch {
      remaining.push(entry)
    }
  }

  writeQueue(remaining)
}

export function setupOfflineQueueSync() {
  window.addEventListener('online', () => {
    void flushQueuedRequests()
  })
}
