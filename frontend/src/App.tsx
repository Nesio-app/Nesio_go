import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import TodayPage from './pages/Today'
import TasksPage from './pages/Tasks'
import ChatPage from './pages/Chat'
import LoginPage from './pages/Login'
import SettingsPage from './pages/Settings'

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/" element={<Layout />}>
        <Route index element={<TodayPage />} />
        <Route path="tasks" element={<TasksPage />} />
        <Route path="chat" element={<ChatPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Route>
    </Routes>
  )
}

export default App
