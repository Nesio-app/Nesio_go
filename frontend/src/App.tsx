import { useState } from 'react'
import TabBar from './components/TabBar'
import TodayPage from './pages/TodayPage'
import ChatPage from './pages/ChatPage'
import MemoryPage from './pages/MemoryPage'
import SettingsPage from './pages/SettingsPage'
import DomainsPage from './pages/DomainsPage'

type Tab = 'today' | 'chat' | 'memory' | 'settings' | 'domains'

export default function App() {
  const [tab, setTab] = useState<Tab>('today')
  const [prevTab, setPrevTab] = useState<Tab>('today')

  const navigate = (t: Tab) => {
    setPrevTab(tab)
    setTab(t)
  }

  const goBack = () => {
    setTab(prevTab)
  }

  // 哪些页面显示 TabBar
  const showTabBar = ['today', 'memory', 'domains'].includes(tab)

  return (
    <div className="h-screen w-full max-w-md mx-auto bg-nesio-bg flex flex-col overflow-hidden relative">
      <main className="flex-1 overflow-y-auto scrollbar-hide">
        {tab === 'today' && (
          <TodayPage
            onMemory={() => navigate('memory')}
            onSettings={() => navigate('settings')}
            onChat={() => navigate('chat')}
          />
        )}
        {tab === 'chat' && <ChatPage onBack={goBack} />}
        {tab === 'memory' && <MemoryPage onBack={() => navigate('today')} />}
        {tab === 'settings' && <SettingsPage onBack={goBack} />}
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
