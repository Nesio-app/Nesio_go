import { useQuery } from '@tanstack/react-query'
import {
  IconSun, IconShield, IconGift, IconHelp, IconBulb, IconChevronRight,
  IconSettings, IconArrowLeft
} from '../icons'
import { connectors as connectorsApi } from '../api/client'

const menuItems = [
  { icon: IconSun, label: '外观与语言' },
  { icon: IconShield, label: '数据与隐私' },
  { icon: IconGift, label: '会员 · Pro' },
  { icon: IconHelp, label: '帮助与反馈' },
  { icon: IconBulb, label: 'Lab' },
]

interface Props {
  onBack: () => void
  themeMode: 'light' | 'dark'
  palette: 'dawn' | 'ocean' | 'forest'
  onThemeChange: (mode: 'light' | 'dark') => void
  onPaletteChange: (palette: 'dawn' | 'ocean' | 'forest') => void
}

export default function SettingsPage({ onBack, themeMode, palette, onThemeChange, onPaletteChange }: Props) {
  const connectorListQuery = useQuery({
    queryKey: ['connectors'],
    queryFn: async () => {
      const r = await connectorsApi.list()
      return r.data as Array<{ id: string; provider: string; is_active: boolean; last_sync_at?: string | null }>
    },
  })
  const gmailConnected = connectorListQuery.data?.some((c) => c.provider === 'gmail' && c.is_active)

  const connectGmail = async () => {
    try {
      const { data } = await (await import('../api/client')).gmail.authorizeUrl()
      window.location.href = data.auth_url as string
    } catch {
      alert('Gmail OAuth 尚未配置：请在服务器设置 GOOGLE_CLIENT_ID 和 GOOGLE_CLIENT_SECRET。')
    }
  }

  return (
    <div className="px-5 pt-6 pb-4 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <button
          onClick={onBack}
          className="w-8 h-8 flex items-center justify-center text-nesio-ink active:scale-90 transition"
        >
          <IconArrowLeft className="w-5 h-5" />
        </button>
        <button className="w-8 h-8 flex items-center justify-center text-nesio-muted active:scale-90 transition">
          <IconSettings className="w-5 h-5" />
        </button>
      </div>

      {/* Profile Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-14 h-14 rounded-full bg-nesio-accentSoft overflow-hidden flex items-center justify-center">
            <span className="text-xl font-bold text-nesio-accent">婧</span>
          </div>
          <span className="text-xl font-bold text-nesio-ink">婧</span>
          <IconChevronRight className="w-5 h-5 text-nesio-muted" />
        </div>
        <button
          onClick={onBack}
          className="text-nesio-accent text-sm font-medium active:opacity-70 transition"
        >
          返回今天
        </button>
      </div>

      {/* Connectors */}
      <div className="nesio-card p-4 space-y-3">
        <div className="text-sm font-semibold text-nesio-ink">连接账号</div>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-nesio-accentSoft flex items-center justify-center">
                <svg className="w-5 h-5 text-nesio-accent" viewBox="0 0 24 24" fill="currentColor">
                <path d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 4l-8 5-8-5V6l8 5 8-5v2z"/>
              </svg>
            </div>
            <div>
              <div className="text-sm font-medium text-nesio-ink">Gmail</div>
              <div className="text-xs text-nesio-muted">
                {gmailConnected ? '已连接 · 可收发邮件' : '未连接'}
              </div>
            </div>
          </div>
          <button
            onClick={connectGmail}
            className={`ui-btn rounded-full px-4 ${
              gmailConnected ? 'bg-nesio-border text-nesio-muted' : 'bg-nesio-accent text-white'
            }`}
          >
            {gmailConnected ? '重新授权' : '连接'}
          </button>
        </div>
      </div>

      <div className="nesio-card p-4 space-y-4">
        <div>
          <div className="text-sm text-nesio-muted mb-2">主题模式</div>
          <div className="flex gap-2">
            {(['light', 'dark'] as const).map((mode) => (
              <button
                key={mode}
                onClick={() => onThemeChange(mode)}
                className={`ui-chip transition ${themeMode === mode ? 'ui-chip-active' : ''}`}
              >
                {mode === 'light' ? '浅色' : '深色'}
              </button>
            ))}
          </div>
        </div>
        <div>
          <div className="text-sm text-nesio-muted mb-2">配色</div>
          <div className="flex gap-2 flex-wrap">
            {([
              { key: 'dawn', label: '晨雾' },
              { key: 'ocean', label: '海盐' },
              { key: 'forest', label: '林地' },
            ] as const).map((item) => (
              <button
                key={item.key}
                onClick={() => onPaletteChange(item.key)}
                className={`ui-chip transition ${palette === item.key ? 'ui-chip-active' : ''}`}
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Menu Items */}
      <div className="space-y-3">
        {menuItems.map((item) => (
          <button
            key={item.label}
            className="w-full ui-card-plain p-4 flex items-center gap-4 active:scale-[0.99] transition"
          >
            <div className="w-12 h-12 rounded-2xl flex items-center justify-center bg-nesio-accentSoft text-nesio-accent">
              <item.icon className="w-6 h-6" />
            </div>
            <span className="flex-1 text-left text-base font-medium text-nesio-ink">{item.label}</span>
            <IconChevronRight className="w-5 h-5 text-nesio-muted" />
          </button>
        ))}
      </div>
    </div>
  )
}
