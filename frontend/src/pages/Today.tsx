import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { signals, today } from '../api/client'

export default function TodayPage() {
  const qc = useQueryClient()
  const nav = useNavigate()
  const [slotFilter, setSlotFilter] = useState('')
  const [minSeverity, setMinSeverity] = useState('0')
  const { data } = useQuery({
    queryKey: ['today', slotFilter, minSeverity],
    queryFn: () => today.get(undefined, slotFilter, minSeverity).then(r => r.data),
  })
  const cards = data?.cards || []

  const [source, setSource] = useState('note')
  const [rawData, setRawData] = useState('')
  const [error, setError] = useState('')

  const dismiss = useMutation({
    mutationFn: (id: string) => today.dismiss(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['today'] }),
  })

  const mute = useMutation({
    mutationFn: (id: string) => today.mute(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['today'] }),
  })

  const done = useMutation({
    mutationFn: (id: string) => today.done(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['today'] }),
  })

  const createSignal = useMutation({
    mutationFn: (payload: any) => signals.create(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['today'] })
      setRawData('')
      setError('')
    },
    onError: (err: any) => {
      setError(err.response?.data?.message || 'Unable to capture')
    },
  })

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-xl bg-white/90 flex items-center justify-center">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="#6B9FD4"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
          </div>
          <span className="text-white font-bold text-lg drop-shadow">Nesio</span>
        </div>
        <button onClick={() => nav('/chat')} className="avatar">J</button>
      </div>

      <div className="card-featured mb-4">
        <div className="flex items-start justify-between mb-3">
          <span className="text-xs font-semibold uppercase tracking-wider opacity-80">NOW</span>
          <div className="w-10 h-10 rounded-full border-2 border-white/60 flex items-center justify-center font-bold text-sm">92</div>
        </div>
        <h2 className="text-xl font-bold mb-2 leading-snug">Put the gray coat by the door</h2>
        <p className="text-sm opacity-90 mb-4 leading-relaxed">Temp drops tomorrow afternoon; your throat is still recovering.</p>
        <div className="flex gap-2">
          <button className="bg-white text-[#6B9FD4] rounded-full px-4 py-2 text-sm font-medium">Okay, at the door</button>
          <button className="bg-white/20 text-white rounded-full px-4 py-2 text-sm font-medium">Why?</button>
        </div>
      </div>

      <div className="card-white mb-4 p-4 rounded-3xl bg-white/95 shadow-sm">
        <div className="flex items-center justify-between mb-3">
          <div>
            <p className="text-xs uppercase tracking-wider text-[#94A3B8]">Capture</p>
            <h2 className="font-semibold text-lg">Quick signal</h2>
          </div>
          <span className="text-xs text-[#6B9FD4]">Live</span>
        </div>
        <div className="space-y-3">
          <div>
            <label className="text-xs uppercase tracking-wider text-[#94A3B8]">Source</label>
            <select value={source} onChange={(e) => setSource(e.target.value)} className="w-full mt-2 rounded-2xl border border-slate-200 px-3 py-3 text-sm outline-none">
              <option value="note">Note</option>
              <option value="email">Email</option>
              <option value="calendar">Calendar</option>
            </select>
          </div>
          <div>
            <label className="text-xs uppercase tracking-wider text-[#94A3B8]">What should Nesio know?</label>
            <textarea
              value={rawData}
              onChange={(e) => setRawData(e.target.value)}
              rows={4}
              className="w-full mt-2 rounded-3xl border border-slate-200 px-4 py-3 text-sm outline-none"
              placeholder="Write a quick reminder, meeting note, or email summary"
            />
          </div>
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            onClick={() => {
              if (!rawData.trim()) {
                setError('Please enter something to capture.')
                return
              }
              createSignal.mutate({
                source,
                anchor_id: `${source}-${Date.now()}`,
                raw_data: rawData,
                fields: { text: rawData },
              })
            }}
            disabled={createSignal.isPending}
            className="w-full rounded-2xl bg-[#6B9FD4] px-4 py-3 text-sm font-semibold text-white disabled:opacity-60"
          >
            {createSignal.isPending ? 'Capturing…' : 'Capture signal'}
          </button>
        </div>
      </div>

      <div className="flex flex-wrap gap-3 mb-4">
        <div className="w-full sm:w-auto">
          <label className="text-xs uppercase tracking-wider text-[#94A3B8]">Slot</label>
          <select value={slotFilter} onChange={(e) => setSlotFilter(e.target.value)} className="w-full mt-2 rounded-2xl border border-slate-200 px-3 py-3 text-sm outline-none">
            <option value="">All slots</option>
            <option value="pinned">Pinned</option>
            <option value="guidance">Guidance</option>
            <option value="task">Task</option>
          </select>
        </div>
        <div className="w-full sm:w-auto">
          <label className="text-xs uppercase tracking-wider text-[#94A3B8]">Severity</label>
          <select value={minSeverity} onChange={(e) => setMinSeverity(e.target.value)} className="w-full mt-2 rounded-2xl border border-slate-200 px-3 py-3 text-sm outline-none">
            <option value="0">All severities</option>
            <option value="1">Normal+</option>
            <option value="2">Important+</option>
            <option value="3">Critical only</option>
          </select>
        </div>
      </div>
      <div className="space-y-3">
        <div className="card-white flex items-center gap-4">
          <div className="w-10 h-10 rounded-xl bg-[#F0F4F8] flex items-center justify-center text-[#6B9FD4]">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/></svg>
          </div>
          <div className="flex-1">
            <p className="text-xs text-[#94A3B8] font-medium uppercase tracking-wider mb-1">AUDIO BRIEF</p>
            <h3 className="font-semibold text-[15px] leading-snug">Tomorrow's meeting: 3 key points organized</h3>
            <p className="text-sm text-[#94A3B8] mt-1">9:10 · Gentle reminder</p>
          </div>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#CBD5E1" strokeWidth="2"><polyline points="9 18 15 12 9 6"/></svg>
        </div>

        <div className="card-white flex items-center gap-4">
          <div className="w-10 h-10 rounded-xl bg-[#F0F4F8] flex items-center justify-center text-emerald-500">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/></svg>
          </div>
          <div className="flex-1">
            <p className="text-xs text-[#94A3B8] font-medium uppercase tracking-wider mb-1">FAMILY FUTURE</p>
            <h3 className="font-semibold text-[15px] leading-snug">Linda's gift is secured</h3>
            <p className="text-sm text-[#94A3B8] mt-1">Storeroom blue box · View in Memory</p>
          </div>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#CBD5E1" strokeWidth="2"><polyline points="9 18 15 12 9 6"/></svg>
        </div>

        {cards.map((card: any) => (
          <div key={card.id} className="card-white rounded-3xl bg-white/95 p-4 shadow-sm">
            <div className="flex flex-col gap-3">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <h3 className="font-semibold text-[15px]">{card.title}</h3>
                    <span className={`text-[11px] rounded-full px-2 py-1 ${card.severity === 3 ? 'bg-red-100 text-red-600' : card.severity === 2 ? 'bg-amber-100 text-amber-700' : 'bg-slate-100 text-slate-600'}`}>
                      {card.severity === 3 ? 'Critical' : card.severity === 2 ? 'Important' : 'Normal'}
                    </span>
                  </div>
                  {card.body && <p className="text-sm text-[#94A3B8]">{card.body}</p>}
                </div>
              </div>
              <div className="flex flex-wrap gap-2 pt-2 border-t border-slate-200">
                <button onClick={() => done.mutate(card.id)} className="rounded-full border border-slate-200 px-3 py-2 text-sm text-emerald-600">Done</button>
                <button onClick={() => mute.mutate(card.id)} className="rounded-full border border-slate-200 px-3 py-2 text-sm text-[#6B9FD4]">Mute</button>
                <button onClick={() => dismiss.mutate(card.id)} className="rounded-full border border-slate-200 px-3 py-2 text-sm text-[#CBD5E1]">Dismiss</button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}