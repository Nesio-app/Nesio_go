import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useState } from 'react'

export default function Layout() {
  const loc = useLocation()
  const nav = useNavigate()
  const [showMenu, setShowMenu] = useState(false)

  return (
    <div className="min-h-screen flex flex-col relative">
      <main className="flex-1 overflow-y-auto pb-24 px-5 pt-6">
        <Outlet />
      </main>

      {showMenu && (
        <div className="fixed inset-0 z-50 flex items-end justify-center pb-28" onClick={() => setShowMenu(false)}>
          <div className="flex gap-4 mb-4" onClick={e => e.stopPropagation()}>
            {['Snap', 'Speak', 'Share'].map((label) => (
              <button key={label} className="flex flex-col items-center gap-2" onClick={() => { nav('/chat'); setShowMenu(false) }}>
                <div className="w-14 h-14 rounded-2xl bg-white shadow-lg flex items-center justify-center text-sm font-bold text-[#6B9FD4]">{label[0]}</div>
                <span className="text-xs text-white font-medium drop-shadow">{label}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      <nav className="bottom-nav fixed bottom-0 left-0 right-0 h-20 flex items-center justify-around px-8 z-40">
        <button onClick={() => nav('/')} className={`flex flex-col items-center gap-1 ${loc.pathname === '/' ? 'text-[#6B9FD4]' : 'text-[#94A3B8]'}`}>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          <span className="text-[11px] font-medium">Today</span>
        </button>

        <button onClick={() => setShowMenu(!showMenu)} className="nav-center">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="white"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
        </button>

        <button onClick={() => nav('/memory')} className={`flex flex-col items-center gap-1 ${loc.pathname === '/memory' ? 'text-[#6B9FD4]' : 'text-[#94A3B8]'}`}>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
          <span className="text-[11px] font-medium">Memory</span>
        </button>
      </nav>
    </div>
  )
}