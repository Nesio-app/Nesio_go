import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

const tags = [
  { label: 'All', count: 3318, active: true },
  { label: 'People', count: 133 },
  { label: 'Items', count: 0 },
]

export default function MemoryPage() {
  const [activeTag, setActiveTag] = useState('All')
  const nav = useNavigate()

  return (
    <div className="min-h-screen px-5 pt-6 pb-24">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-white font-bold text-xl drop-shadow">Memory</h1>
        <button onClick={() => nav('/chat')} className="avatar">J</button>
      </div>

      <div className="bg-white/90 backdrop-blur rounded-2xl flex items-center gap-3 px-4 py-3 mb-6 shadow-sm">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#94A3B8" strokeWidth="2">
          <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        <span className="text-[#94A3B8] text-sm">Search memory</span>
      </div>

      <div className="grid grid-cols-3 gap-3 mb-6">
        {[
          { icon: 'B', label: 'Bookmarks', sub: '1 saved' },
          { icon: 'S', label: 'Storage', sub: '10 items' },
          { icon: 'P', label: 'Projects', sub: '1 active' },
        ].map((item, i) => (
          <div key={i} className="bg-white rounded-2xl p-4 flex flex-col items-center shadow-sm">
            <div className="w-10 h-10 rounded-full bg-[#F0F4F8] flex items-center justify-center text-[#6B9FD4] font-bold mb-2">{item.icon}</div>
            <span className="font-semibold text-sm">{item.label}</span>
            <span className="text-xs text-[#94A3B8] mt-1">{item.sub}</span>
          </div>
        ))}
      </div>

      <h2 className="text-lg font-bold mb-3 text-white drop-shadow">All memories · 3318</h2>

      <div className="flex gap-2 mb-4 overflow-x-auto">
        {tags.map((t) => (
          <button key={t.label} onClick={() => setActiveTag(t.label)}
            className={`px-4 py-2 rounded-xl text-sm font-medium whitespace-nowrap ${activeTag === t.label ? 'bg-white text-[#6B9FD4]' : 'bg-white/30 text-white'}`}>
            {t.label} <span className="ml-1 opacity-70">{t.count}</span>
          </button>
        ))}
      </div>
    </div>
  )
}