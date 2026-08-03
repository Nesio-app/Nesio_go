import { FormEvent, useState } from 'react'
import { auth } from '../api/client'

interface Props {
  onAuthenticated: () => void
}

export default function AuthPage({ onAuthenticated }: Props) {
  const [mode, setMode] = useState<'login' | 'register' | 'forgot'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [resetToken, setResetToken] = useState('')
  const [resetExpiresAt, setResetExpiresAt] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!email.trim()) {
      setError('请输入邮箱。')
      return
    }

    if (mode !== 'forgot' && password.length < 6) {
      setError('请输入邮箱和至少 6 位密码。')
      return
    }

    if (mode === 'forgot' && !resetToken.trim()) {
      setError('请先生成重置令牌。')
      return
    }

    setIsSubmitting(true)
    setError('')
    setNotice('')
    try {
      if (mode === 'login') {
        const response = await auth.login(email.trim(), password)
        localStorage.setItem('token', response.data.token as string)
        onAuthenticated()
        return
      }

      if (mode === 'register') {
        const response = await auth.register(email.trim(), password)
        localStorage.setItem('token', response.data.token as string)
        onAuthenticated()
        return
      }

      await auth.resetPassword(email.trim(), resetToken.trim(), password)
      setNotice('密码已重置，请返回登录。')
      setMode('login')
      setPassword('')
      setResetToken('')
    } catch {
      if (mode === 'login') {
        setError('邮箱或密码不正确。')
      } else if (mode === 'register') {
        setError('注册失败，该邮箱可能已存在。')
      } else {
        setError('重置失败，请检查邮箱和令牌。')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  const requestResetToken = async () => {
    if (!email.trim()) {
      setError('请输入邮箱。')
      return
    }

    setIsSubmitting(true)
    setError('')
    setNotice('')
    try {
      const response = await auth.forgotPassword(email.trim())
      setResetToken(response.data.reset_token)
      setResetExpiresAt(response.data.expires_at)
      setMode('forgot')
      setNotice('已生成重置令牌，请把令牌填入下面的输入框。')
    } catch {
      setError('没有找到这个邮箱对应的账号。')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main className="h-full bg-nesio-bg px-5 py-8 flex items-center justify-center">
      <section className="w-full max-w-sm">
        <div className="mb-8">
          <div className="w-12 h-12 rounded-xl bg-nesio-accent mb-5" />
          <h1 className="type-display text-nesio-ink">Nesio</h1>
          <p className="mt-2 type-body text-nesio-muted">登录后，你的任务和记忆会保存到云端。</p>
        </div>

        <form onSubmit={submit} className="space-y-4">
          <label className="block">
            <span className="type-caption text-nesio-muted">邮箱</span>
            <input
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              className="ui-input mt-1"
            />
          </label>
          <label className="block">
            <span className="type-caption text-nesio-muted">密码</span>
            <input
              type="password"
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              className="ui-input mt-1"
              placeholder={mode === 'forgot' ? '输入新密码' : ''}
            />
          </label>

          {mode === 'forgot' && (
            <label className="block">
              <span className="type-caption text-nesio-muted">重置令牌</span>
              <input
                type="text"
                value={resetToken}
                onChange={(event) => setResetToken(event.target.value)}
                className="ui-input mt-1 font-mono text-sm"
                placeholder="点击下方按钮生成"
              />
            </label>
          )}

          {error && <p className="ui-state-error">{error}</p>}
          {notice && <p className="ui-state-success">{notice}</p>}

          {mode === 'forgot' && resetExpiresAt && (
            <p className="type-caption text-nesio-muted">令牌有效期到 {resetExpiresAt}</p>
          )}

          <button
            type="submit"
            disabled={isSubmitting}
            className="ui-btn-primary w-full"
          >
            {isSubmitting ? '请稍候...' : mode === 'login' ? '登录' : mode === 'register' ? '创建账号' : '重置密码'}
          </button>

          {mode === 'forgot' && (
            <button
              type="button"
              onClick={requestResetToken}
              disabled={isSubmitting}
              className="ui-btn-secondary w-full"
            >
              生成重置令牌
            </button>
          )}
        </form>

        <div className="mt-5 flex flex-col gap-2">
          <button
            type="button"
            onClick={() => {
              setMode((current) => current === 'login' ? 'register' : 'login')
              setError('')
              setNotice('')
            }}
            className="ui-btn-link"
          >
            {mode === 'login' ? '没有账号？立即注册' : '已有账号？返回登录'}
          </button>
          {mode !== 'forgot' && (
            <button
              type="button"
              onClick={() => {
                setMode('forgot')
                setError('')
                setNotice('')
              }}
              className="ui-btn-link"
            >
              忘记密码？
            </button>
          )}
          {mode === 'forgot' && (
            <button
              type="button"
              onClick={() => {
                setMode('login')
                setError('')
                setNotice('')
              }}
              className="ui-btn-link"
            >
              返回登录
            </button>
          )}
        </div>
      </section>
    </main>
  )
}