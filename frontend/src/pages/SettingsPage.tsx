import {
  IconSun, IconShield, IconGift, IconHelp, IconBulb, IconChevronRight,
  IconSettings, IconArrowLeft
} from '../icons'

const menuItems = [
  { icon: IconSun, label: '外观与语言', color: 'bg-blue-50 text-blue-500' },
  { icon: IconShield, label: '数据与隐私', color: 'bg-red-50 text-red-400' },
  { icon: IconGift, label: '会员 · Pro', color: 'bg-amber-50 text-amber-500' },
  { icon: IconHelp, label: '帮助与反馈', color: 'bg-indigo-50 text-indigo-500' },
  { icon: IconBulb, label: 'Lab', color: 'bg-purple-50 text-purple-500' },
]

interface Props {
  onBack: () => void
}

export default function SettingsPage({ onBack }: Props) {
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

      {/* Menu Items */}
      <div className="space-y-3">
        {menuItems.map((item) => (
          <button
            key={item.label}
            className="w-full nesio-card p-4 flex items-center gap-4 active:scale-[0.99] transition"
          >
            <div className={`w-12 h-12 rounded-2xl flex items-center justify-center ${item.color}`}>
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
