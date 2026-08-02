import { useEffect, useState } from 'react'
import TabBar from './components/TabBar'
import TodayPage from './pages/TodayPage'
import ChatPage from './pages/ChatPage'
import MemoryPage from './pages/MemoryPage'
import SettingsPage from './pages/SettingsPage'
import DomainsPage from './pages/DomainsPage'
import AuthPage from './pages/AuthPage'

type Tab = 'today' | 'chat' | 'memory' | 'settings' | 'domains'
type ThemeMode = 'light' | 'dark'
type Palette = 'dawn' | 'ocean' | 'forest'

export default function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(() => Boolean(localStorage.getItem('token')))
  const [tab, setTab] = useState<Tab>('today')
  const [prevTab, setPrevTab] = useState<Tab>('today')
  const [themeMode, setThemeMode] = useState<ThemeMode>(() => (localStorage.getItem('nesio-theme-mode') as ThemeMode) || 'light')
  const [palette, setPalette] = useState<Palette>(() => (localStorage.getItem('nesio-palette') as Palette) || 'dawn')

  useEffect(() => {
    const handleUnauthorized = () => setIsAuthenticated(false)
    window.addEventListener('nesio:unauthorized', handleUnauthorized)
    return () => window.removeEventListener('nesio:unauthorized', handleUnauthorized)
  }, [])

  useEffect(() => {
    document.documentElement.dataset.theme = themeMode
    document.documentElement.dataset.palette = palette
    localStorage.setItem('nesio-theme-mode', themeMode)
    localStorage.setItem('nesio-palette', palette)
  }, [themeMode, palette])

  const navigate = (t: Tab) => {
    setPrevTab(tab)
    setTab(t)
  }

  const goBack = () => {
    setTab(prevTab)
  }

  // 哪些页面显示 TabBar
  const showTabBar = ['today', 'memory', 'domains'].includes(tab)

  if (!isAuthenticated) {
    return <AuthPage onAuthenticated={() => setIsAuthenticated(true)} />
  }

  return (
    <div className="h-screen w-full max-w-md mx-auto bg-nesio-bg flex flex-col overflow-hidden relative">
      <main key={tab} className="page-shell flex-1 overflow-y-auto scrollbar-hide">
        {tab === 'today' && (
          <TodayPage
            onMemory={() => navigate('memory')}
            onSettings={() => navigate('settings')}
            onChat={() => navigate('chat')}
          />
        )}
        {tab === 'chat' && <ChatPage onBack={goBack} />}
        {tab === 'memory' && <MemoryPage onBack={() => navigate('today')} />}
        {tab === 'settings' && (
          <SettingsPage
            onBack={goBack}
            themeMode={themeMode}
            palette={palette}
            onThemeChange={setThemeMode}
            onPaletteChange={setPalette}
          />
        )}
        {tab === 'domains' && <DomainsPage />}
      </main>
      {showTabBar && (
        <TabBar
          active={tab}
          onChange={(t) => {
            if (t === 'chat') navigate('chat')
            else navigate(t as Tab)
          }}
        />
      )}
    </div>
  )
}
