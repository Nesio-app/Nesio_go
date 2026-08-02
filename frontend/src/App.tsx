import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import TodayPage from './pages/Today'
import MemoryPage from './pages/Memory'
import ConnectorsPage from './pages/Connectors'
import ChatPage from './pages/Chat'
import LoginPage from './pages/Login'

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/" element={<Layout />}>
        <Route index element={<TodayPage />} />
        <Route path="memory" element={<MemoryPage />} />
        <Route path="connectors" element={<ConnectorsPage />} />
        <Route path="chat" element={<ChatPage />} />
      </Route>
    </Routes>
  )
}
export default App