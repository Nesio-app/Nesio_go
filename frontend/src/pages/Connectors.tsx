import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { connectors } from '../api/client'

export default function ConnectorsPage() {
  const nav = useNavigate()
  const qc = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['connectors'], queryFn: () => connectors.list().then(r => r.data) })
  const [provider, setProvider] = useState('gmail')
  const [token, setToken] = useState('')
  const [error, setError] = useState('')

  const create = useMutation({
    mutationFn: (payload: any) => connectors.auth(payload.provider, payload.credentials),
    onSuccess: (result: any) => {
      const res = result?.data
      if (res?.auth_url) {
        window.open(res.auth_url, '_blank')
      }
      qc.invalidateQueries({ queryKey: ['connectors'] })
      setToken('')
      setError('')
    },
    onError: (err: any) => setError(err.response?.data?.message || 'Unable to connect'),
  })

  const sync = useMutation({
    mutationFn: (id: string) => connectors.sync(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['connectors'] }),
  })

  const remove = useMutation({
    mutationFn: (id: string) => connectors.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['connectors'] }),
  })

  return (
    <div className="min-h-screen px-5 pt-6 pb-24">
      <div className="flex items-center justify-between mb-6">
        <button onClick={() => nav(-1)} className="text-white">Back</button>
        <h1 className="text-white font-bold text-xl drop-shadow">Connectors</h1>
        <div className="w-8 h-8" />
      </div>

      <div className="bg-white/90 rounded-3xl p-5 mb-6 shadow-sm">
        <div className="flex items-center justify-between mb-4">
          <div>
            <p className="text-xs uppercase tracking-wider text-[#94A3B8]">Add connector</p>
            <h2 className="font-semibold text-lg">Secure credential sync</h2>
          </div>
        </div>

        <div className="space-y-3">
          <select value={provider} onChange={(e) => setProvider(e.target.value)} className="w-full rounded-2xl border border-slate-200 px-4 py-3 text-sm outline-none">
            <option value="gmail">Gmail</option>
            <option value="calendar">Calendar</option>
            <option value="plaid">Plaid</option>
          </select>
          <textarea
            value={token}
            onChange={(e) => setToken(e.target.value)}
            rows={4}
            placeholder="Enter connector credentials JSON"
            className="w-full rounded-3xl border border-slate-200 px-4 py-3 text-sm outline-none"
          />
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            onClick={() => {
              if (!token.trim()) {
                create.mutate({ provider, credentials: {} })
                return
              }
              try {
                const credentials = JSON.parse(token)
                create.mutate({ provider, credentials })
              } catch (e) {
                setError('Invalid JSON')
              }
            }}
            className="w-full rounded-2xl bg-[#6B9FD4] px-4 py-3 text-sm font-semibold text-white"
          >
            Connect {provider}
          </button>
          <p className="text-xs text-slate-500 mt-2">Leave credentials blank to start the OAuth-style flow.</p>
        </div>
      </div>

      <div className="space-y-3">
        {isLoading && <div className="text-white/80">Loading connectors…</div>}
        {data?.length === 0 && !isLoading && <div className="text-white/80">No connectors configured yet.</div>}
        {data?.map((item: any) => (
          <div key={item.id} className="bg-white rounded-3xl p-4 shadow-sm flex items-center justify-between">
            <div>
              <div className="text-sm font-semibold text-slate-900">{item.provider}</div>
              <div className="text-xs text-slate-500">{item.is_active ? 'Connected' : 'Inactive'}</div>
            </div>
            <div className="flex gap-2">
              <button onClick={() => sync.mutate(item.id)} className="rounded-full border border-slate-200 px-3 py-2 text-sm text-[#6B9FD4]">Sync</button>
              <button onClick={() => remove.mutate(item.id)} className="rounded-full border border-slate-200 px-3 py-2 text-sm">Remove</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
