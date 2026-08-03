import { useEffect, useState } from 'react'
import TabBar from './components/TabBar'
import TodayPage from './pages/TodayPage'
import ChatPage from './pages/ChatPage'
import MemoryPage from './pages/MemoryPage'
import SettingsPage from './pages/SettingsPage'
import DomainsPage from './pages/DomainsPage'
import ItemsPage from './pages/ItemsPage'
import AuthPage from './pages/AuthPage'
import CapturePage from './pages/CapturePage'
import RecognitionResultPage from './pages/RecognitionResultPage'
import ItemDetailPage from './pages/ItemDetailPage'

type Tab = 'today' | 'chat' | 'memory' | 'settings' | 'domains' | 'capture' | 'recognition' | 'items' | 'item-detail'
type ThemeMode = 'light' | 'dark'
type Palette = 'dawn' | 'ocean' | 'forest'

export default function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(() => Boolean(localStorage.getItem('token')))
  const [tab, setTab] = useState<Tab>('today')
  const [prevTab, setPrevTab] = useState<Tab>('today')
  const [captureRequestToken, setCaptureRequestToken] = useState(0)
  const [selectedItemId, setSelectedItemId] = useState<string | null>(null)
  const [capturePayload, setCapturePayload] = useState<{
    file: File
    previewUrl: string
    result: {
      extraction: Record<string, any>
      duplicates: Array<Record<string, any>>
      visual_hash: string
    }
  } | null>(null)
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

  const tabBarActiveTab = tab === 'today' ? 'today' : (tab === 'domains' || tab === 'items' || tab === 'item-detail' ? 'domains' : 'today')

  // 哪些页面显示 TabBar
  const showTabBar = ['today', 'memory', 'domains', 'items'].includes(tab)

  if (!isAuthenticated) {
    return (
      <div className="app-frame bg-nesio-bg flex flex-col relative">
        <main className="page-shell app-content-safe flex-1 overflow-y-auto scrollbar-hide">
          <AuthPage onAuthenticated={() => setIsAuthenticated(true)} />
        </main>
      </div>
    )
  }

  return (
    <div className="app-frame bg-nesio-bg flex flex-col relative">
      <main key={tab} className="page-shell app-content-safe flex-1 overflow-y-auto scrollbar-hide">
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
        {tab === 'domains' && <DomainsPage onToday={() => navigate('today')} onOpenItems={() => navigate('items')} />}
        {tab === 'items' && (
          <ItemsPage
            onOpenItem={(itemId) => {
              setSelectedItemId(itemId)
              setPrevTab('items')
              setTab('item-detail')
            }}
          />
        )}
        {tab === 'item-detail' && selectedItemId && (
          <ItemDetailPage
            itemId={selectedItemId}
            onBack={() => {
              setTab('items')
            }}
          />
        )}
        {tab === 'capture' && (
          <CapturePage
            onClose={() => setTab(prevTab)}
            captureRequestToken={captureRequestToken}
            onAnalyzed={(payload) => {
              setCapturePayload(payload)
              setTab('recognition')
            }}
          />
        )}
        {tab === 'recognition' && capturePayload && (
          <RecognitionResultPage
            previewUrl={capturePayload.previewUrl}
            result={capturePayload.result}
            onClose={() => {
              setCapturePayload(null)
              setTab('items')
            }}
          />
        )}
      </main>
      {showTabBar && (
        <TabBar
          active={tabBarActiveTab}
          onCameraPress={() => {
            setPrevTab(tab)
            setCaptureRequestToken((current) => current + 1)
            setTab('capture')
          }}
          onAskPress={() => {
            setPrevTab(tab)
            setTab('chat')
          }}
          onChange={(t) => {
            navigate(t as Tab)
          }}
        />
      )}
    </div>
  )
}
