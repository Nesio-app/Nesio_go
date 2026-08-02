import { useQuery } from '@tanstack/react-query'
import { user } from '../api/client'

export default function SettingsPage() {
  const { data } = useQuery({
    queryKey: ['me'],
    queryFn: () => user.me().then((r) => r.data),
  })

  return (
    <div>
      <h2 className="text-xl font-bold mb-6">设置</h2>
      <div className="card space-y-4">
        <div>
          <label className="text-sm text-[var(--color-text-secondary)]">邮箱</label>
          <p className="font-medium">{data?.email || '...'}</p>
        </div>
        <div>
          <label className="text-sm text-[var(--color-text-secondary)]">时区</label>
          <p className="font-medium">{data?.timezone || 'Asia/Shanghai'}</p>
        </div>
        <div>
          <label className="text-sm text-[var(--color-text-secondary)]">语言</label>
          <p className="font-medium">{data?.locale === 'zh' ? '中文' : data?.locale || '中文'}</p>
        </div>
        <hr className="border-[var(--color-border)]" />
        <button
          onClick={() => {
            localStorage.removeItem('token')
            window.location.href = '/login'
          }}
          className="w-full py-2 text-[var(--color-status-risk)] font-medium"
        >
          退出登录
        </button>
      </div>
    </div>
  )
}
