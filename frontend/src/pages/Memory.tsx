import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { memories } from '../api/client'

export default function MemoryPage() {
  const [activeTag, setActiveTag] = useState('All')
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [error, setError] = useState('')
  const nav = useNavigate()
  const qc = useQueryClient()

  const { data } = useQuery({ queryKey: ['memories'], queryFn: () => memories.list().then(r => r.data) })
  const memoryList = data || []

  const createMemory = useMutation({
    mutationFn: (payload: any) => memories.create(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['memories'] })
      setTitle('')
      setBody('')
      setError('')
    },
    onError: (err: any) => setError(err.response?.data?.message || 'Unable to save memory'),
  })

  return (
    <div className="min-h-screen px-5 pt-6 pb-24">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-white font-bold text-xl drop-shadow">Memory</h1>
        <button onClick={() => nav('/chat')} className="w-8 h-8 rounded-full bg-white flex items-center justify-center text-sm font-bold text-[#6B9FD4]">J</button>
      </div>

      <div className="bg-white/90 rounded-2xl px-4 py-4 mb-6 shadow-sm">
        <div className="flex items-center justify-between mb-3">
          <div>
            <p className="text-xs uppercase tracking-wider text-[#94A3B8]">Memory capture</p>
            <h2 className="font-semibold text-lg">Save a memory</h2>
          </div>
          <button onClick={() => nav('/connectors')} className="text-sm font-semibold text-[#6B9FD4]">Manage connectors</button>
        </div>
        <div className="space-y-3">
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Memory title"
            className="w-full rounded-2xl border border-slate-200 px-4 py-3 text-sm outline-none"
          />
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            rows={4}
            placeholder="Details, context, or why this matters"
            className="w-full rounded-3xl border border-slate-200 px-4 py-3 text-sm outline-none"
          />
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            onClick={() => {
              if (!title.trim()) {
                setError('Title is required')
                return
              }
              createMemory.mutate({ title, body, tags: [] })
            }}
            disabled={createMemory.isPending}
            className="w-full rounded-2xl bg-[#6B9FD4] px-4 py-3 text-sm font-semibold text-white disabled:opacity-60"
          >
            {createMemory.isPending ? 'Saving…' : 'Save memory'}
          </button>
        </div>
      </div>

      <h2 className="text-lg font-bold mb-3 text-white drop-shadow">All memories</h2>

      <div className="flex gap-2 mb-4">
        {['All', 'People', 'Items'].map((t) => (
          <button key={t} onClick={() => setActiveTag(t)}
            className={`px-4 py-2 rounded-xl text-sm font-medium ${activeTag === t ? 'bg-white text-[#6B9FD4]' : 'bg-white/30 text-white'}`}>
            {t}
          </button>
        ))}
      </div>

      <div className="space-y-3">
        {memoryList.length === 0 && (
          <div className="text-white/80">No memories yet. Capture one to build your timeline.</div>
        )}
        {memoryList.map((item: any) => (
          <div key={item.id} className="bg-white rounded-3xl p-4 shadow-sm">
            <div className="flex items-center justify-between mb-2">
              <h3 className="font-semibold text-slate-900">{item.title}</h3>
              <span className="text-xs text-[#94A3B8]">{new Date(item.created_at).toLocaleDateString()}</span>
            </div>
            {item.body && <p className="text-sm text-slate-600">{item.body}</p>}
          </div>
        ))}
      </div>
    </div>
  )
}
