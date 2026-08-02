import { IconClock, IconDiamond, IconHexagon } from '../icons'

type Tab = 'today' | 'chat' | 'memory' | 'settings' | 'domains'

interface Props {
  active: Tab
  onChange: (t: Tab) => void
}

export default function TabBar({ active, onChange }: Props) {
  return (
    <nav className="bg-nesio-tabBar border-t border-nesio-border px-6 pb-6 pt-3 flex items-center justify-between shrink-0">
      <button
        onClick={() => onChange('today')}
        className={`flex flex-col items-center gap-1 transition active:scale-95 ${active === 'today' ? 'text-nesio-accent' : 'text-nesio-muted'}`}
      >
        <IconClock className="w-6 h-6" />
        <span className="text-[11px]">今天</span>
      </button>

      <button
        onClick={() => onChange('chat')}
        className="relative -mt-8 active:scale-95 transition"
      >
        <div className="w-16 h-16 rounded-full bg-nesio-accent flex items-center justify-center shadow-float">
          <IconHexagon className="w-8 h-8 text-white" />
        </div>
      </button>

      {/* 右下角"洞察" - 打开领域页 */}
      <button
        onClick={() => onChange('domains')}
        className={`flex flex-col items-center gap-1 transition active:scale-95 ${active === 'domains' ? 'text-nesio-accent' : 'text-nesio-muted'}`}
      >
        <IconDiamond className="w-6 h-6" />
        <span className="text-[11px]">洞察</span>
      </button>
    </nav>
  )
}
