import { FormEvent, useState } from 'react'
import { auth } from '../api/client'

interface Props {
  onAuthenticated: () => void
}

export default function AuthPage({ onAuthenticated }: Props) {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!email.trim() || password.length < 6) {
      setError('请输入邮箱和至少 6 位密码。')
      return
    }

    setIsSubmitting(true)
    setError('')
    try {
      const response = mode === 'login'
        ? await auth.login(email.trim(), password)
        : await auth.register(email.trim(), password)
      localStorage.setItem('token', response.data.token as string)
      onAuthenticated()
    } catch {
      setError(mode === 'login' ? '邮箱或密码不正确。' : '注册失败，该邮箱可能已存在。')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main className="min-h-screen bg-nesio-bg px-5 py-10 flex items-center justify-center">
      <section className="w-full max-w-sm">
        <div className="mb-8">
          <div className="w-12 h-12 rounded-xl bg-nesio-accent mb-5" />
          <h1 className="text-3xl font-bold text-nesio-ink">Nesio</h1>
          <p className="mt-2 text-sm text-nesio-muted">登录后，你的任务和记忆会保存到云端。</p>
        </div>

        <form onSubmit={submit} className="space-y-4">
          <label className="block">
            <span className="text-sm text-nesio-muted">邮箱</span>
            <input
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              className="mt-1 w-full rounded-xl border border-nesio-border bg-white px-4 py-3 text-base text-nesio-ink outline-none focus:border-nesio-accent"
            />
          </label>
          <label className="block">
            <span className="text-sm text-nesio-muted">密码</span>
            <input
              type="password"
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              className="mt-1 w-full rounded-xl border border-nesio-border bg-white px-4 py-3 text-base text-nesio-ink outline-none focus:border-nesio-accent"
            />
          </label>

          {error && <p className="text-sm text-red-600">{error}</p>}

          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full rounded-xl bg-nesio-accent px-4 py-3 font-medium text-white disabled:opacity-50"
          >
            {isSubmitting ? '请稍候...' : mode === 'login' ? '登录' : '创建账号'}
          </button>
        </form>

        <button
          type="button"
          onClick={() => {
            setMode((current) => current === 'login' ? 'register' : 'login')
            setError('')
          }}
          className="mt-5 text-sm text-nesio-accent"
        >
          {mode === 'login' ? '没有账号？立即注册' : '已有账号？返回登录'}
        </button>
      </section>
    </main>
  )
}