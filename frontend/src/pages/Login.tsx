import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { auth } from '../api/client'

export default function LoginPage() {
  const [isLogin, setIsLogin] = useState(true)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const nav = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      const fn = isLogin ? auth.login : auth.register
      const { data } = await fn(email, password)
      localStorage.setItem('token', data.token)
      nav('/')
    } catch (err: any) {
      setError(err.response?.data?.message || '请求失败')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-bold text-center mb-8 text-[var(--color-portal-ink)]">
          Nesio
        </h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          <input
            type="email"
            placeholder="邮箱"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full px-4 py-3 rounded-xl border border-[var(--color-border)] bg-white focus:outline-none focus:ring-2 focus:ring-[var(--color-portal-blue)]"
            required
          />
          <input
            type="password"
            placeholder="密码"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full px-4 py-3 rounded-xl border border-[var(--color-border)] bg-white focus:outline-none focus:ring-2 focus:ring-[var(--color-portal-blue)]"
            required
          />
          {error && <p className="text-sm text-[var(--color-status-risk)]">{error}</p>}
          <button type="submit" className="w-full btn-primary py-3">
            {isLogin ? '登录' : '注册'}
          </button>
        </form>
        <p className="text-center mt-4 text-sm text-[var(--color-text-secondary)]">
          {isLogin ? '没有账号？' : '已有账号？'}
          <button
            onClick={() => setIsLogin(!isLogin)}
            className="text-[var(--color-portal-blue)] ml-1 font-medium"
          >
            {isLogin ? '注册' : '登录'}
          </button>
        </p>
      </div>
    </div>
  )
}
