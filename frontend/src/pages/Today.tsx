import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { today } from '../api/client'

export default function TodayPage() {
  const qc = useQueryClient()
  const nav = useNavigate()
  const { data } = useQuery({ queryKey: ['today'], queryFn: () => today.get().then(r => r.data) })
  const cards = data?.cards || []

  const dismiss = useMutation({
    mutationFn: (id: string) => today.dismiss(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['today'] }),
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
          <div key={card.id} className="card-white flex items-center justify-between">
            <div>
              <h3 className="font-semibold text-[15px]">{card.title}</h3>
              {card.body && <p className="text-sm text-[#94A3B8] mt-1">{card.body}</p>}
            </div>
            <button onClick={() => dismiss.mutate(card.id)} className="text-[#CBD5E1]">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}