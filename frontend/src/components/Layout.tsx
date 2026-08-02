import { Outlet, Link, useLocation } from 'react-router-dom'

export default function Layout() {
  const loc = useLocation()
  const nav = [
    { path: '/', label: '今日' },
    { path: '/tasks', label: '任务' },
    { path: '/chat', label: '念念' },
    { path: '/settings', label: '设置' },
  ]

  return (
    <div className="min-h-screen flex flex-col">
      <header className="bg-white border-b border-[var(--color-border)] sticky top-0 z-10">
        <div className="max-w-2xl mx-auto px-4 h-14 flex items-center justify-between">
          <h1 className="text-lg font-bold text-[var(--color-portal-blue)]">Nesio</h1>
          <nav className="flex gap-1">
            {nav.map((n) => (
              <Link
                key={n.path}
                to={n.path}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                  loc.pathname === n.path
                    ? 'bg-[var(--color-portal-blue)] text-white'
                    : 'text-[var(--color-text-secondary)] hover:bg-gray-100'
                }`}
              >
                {n.label}
              </Link>
            ))}
          </nav>
        </div>
      </header>
      <main className="flex-1 max-w-2xl mx-auto w-full px-4 py-6">
        <Outlet />
      </main>
    </div>
  )
}
