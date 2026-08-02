import {
  IconHeart, IconBox, IconCreditCard, IconCalendar,
  IconMapPin, IconActivity, IconUsers, IconHanger,
  IconUser, IconTarget, IconUtensils, IconSparkles,
  IconHome, IconBulb, IconPlay, IconSettings,
  IconMusic, IconGift
} from '../icons'

const domains = [
  { icon: IconHeart, label: '健康', color: 'text-red-400' },
  { icon: IconBox, label: '物品', color: 'text-nesio-accent' },
  { icon: IconCreditCard, label: '财务', color: 'text-nesio-accent' },
  { icon: IconCalendar, label: '日程', color: 'text-nesio-accent' },
  { icon: IconMapPin, label: '足迹', color: 'text-nesio-accent' },
  { icon: IconActivity, label: '健身', color: 'text-nesio-accent' },
  { icon: IconUsers, label: '家务', color: 'text-nesio-accent' },
  { icon: IconHanger, label: '衣橱', color: 'text-nesio-accent' },
  { icon: IconUser, label: '关系', color: 'text-nesio-accent' },
  { icon: IconTarget, label: '成长', color: 'text-nesio-accent' },
  { icon: IconUtensils, label: '美味', color: 'text-nesio-accent' },
  { icon: IconSparkles, label: '镜子', color: 'text-nesio-accent' },
  { icon: IconHome, label: '资产', color: 'text-nesio-accent' },
  { icon: IconBulb, label: '目标', color: 'text-nesio-accent' },
  { icon: IconPlay, label: '剧场', color: 'text-nesio-accent' },
  { icon: IconSettings, label: '运营', color: 'text-nesio-accent' },
  { icon: IconMusic, label: '音乐', color: 'text-nesio-accent' },
  { icon: IconGift, label: '奖励', color: 'text-nesio-accent' },
]

interface Props {
  onToday?: () => void
  onMemory?: () => void
  onChat?: () => void
}

export default function DomainsPage({ onToday, onMemory, onChat }: Props) {
  // avoid unused parameter errors
  void onToday
  void onMemory
  void onChat
  return (
    <div className="px-5 pt-6 pb-4">
      <div className="grid grid-cols-4 gap-3">
        {domains.map((d) => (
          <button
            key={d.label}
            className="flex flex-col items-center gap-2 py-3 active:scale-95 transition"
          >
            <div className="w-16 h-16 rounded-2xl bg-white shadow-card flex items-center justify-center">
              <d.icon className={`w-7 h-7 ${d.color}`} />
            </div>
            <span className="text-sm text-nesio-ink font-medium">{d.label}</span>
          </button>
        ))}
      </div>
    </div>
  )
}
