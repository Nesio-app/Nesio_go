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
      setError(err.response?.data?.message || 'Failed')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center px-6" style={{background: 'linear-gradient(180deg, #7BA7E1, #A8C8F0)'}}>
      <div className="w-full max-w-sm">
        <div className="flex justify-center mb-8">
          <div className="w-20 h-20 rounded-3xl bg-white/90 flex items-center justify-center shadow-lg">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="#6B9FD4"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
          </div>
        </div>
        <h1 className="text-3xl font-bold text-center mb-2 text-white drop-shadow">Nesio</h1>
        <p className="text-center text-white/80 mb-8 text-sm">Less tracking. More living.</p>
        <form onSubmit={handleSubmit} className="space-y-4">
          <input type="email" placeholder="Email" value={email} onChange={(e) => setEmail(e.target.value)} className="w-full px-5 py-4 rounded-2xl bg-white/95 border-none outline-none text-[15px] shadow-sm" required />
          <input type="password" placeholder="Password" value={password} onChange={(e) => setPassword(e.target.value)} className="w-full px-5 py-4 rounded-2xl bg-white/95 border-none outline-none text-[15px] shadow-sm" required />
          {error && <p className="text-sm text-red-200">{error}</p>}
          <button type="submit" className="w-full py-4 rounded-2xl bg-white text-[#6B9FD4] font-bold text-[15px] shadow-lg">{isLogin ? 'Sign In' : 'Sign Up'}</button>
        </form>
        <p className="text-center mt-6 text-sm text-white/80">
          {isLogin ? 'No account?' : 'Have account?'}
          <button onClick={() => setIsLogin(!isLogin)} className="text-white ml-1 font-bold underline">{isLogin ? 'Sign Up' : 'Sign In'}</button>
        </p>
      </div>
    </div>
  )
}