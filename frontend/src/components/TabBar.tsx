import { useRef } from 'react'
import { IconBox, IconClock, IconHexagon } from '../icons'

type Tab = 'today' | 'chat' | 'memory' | 'settings' | 'domains' | 'capture' | 'recognition' | 'items'

interface Props {
  active: Tab
  onChange: (t: Tab) => void
  onCameraPress: () => void
  onAskPress: () => void
}

export default function TabBar({ active, onChange, onCameraPress, onAskPress }: Props) {
  const timerRef = useRef<number | null>(null)
  const longPressFiredRef = useRef(false)
  const isPressingRef = useRef(false)

  const onCenterDown = () => {
    isPressingRef.current = true
    longPressFiredRef.current = false
    timerRef.current = window.setTimeout(() => {
      if (!isPressingRef.current) {
        return
      }
      longPressFiredRef.current = true
      onAskPress()
    }, 650)
  }

  const onCenterUp = () => {
    isPressingRef.current = false
    if (timerRef.current) {
      window.clearTimeout(timerRef.current)
      timerRef.current = null
    }
    if (!longPressFiredRef.current) {
      onCameraPress()
    }
    longPressFiredRef.current = false
  }

  return (
    <nav className="tabbar-safe bg-nesio-tabBar border-t border-nesio-border px-6 pt-3 flex items-center justify-between shrink-0">
      <button
        onClick={() => onChange('today')}
        className={`flex flex-col items-center gap-1 transition active:scale-95 ${active === 'today' ? 'text-nesio-accent' : 'text-nesio-muted'}`}
      >
        <IconClock className="w-6 h-6" />
        <span className="type-caption">今天</span>
      </button>

      <button
        onPointerDown={onCenterDown}
        onPointerUp={onCenterUp}
        onPointerLeave={onCenterUp}
        onPointerCancel={onCenterUp}
        className="relative -mt-8 active:scale-95 transition"
      >
        <div className="w-16 h-16 rounded-full bg-nesio-accent flex items-center justify-center shadow-float">
          <IconHexagon className="w-8 h-8 text-white" />
        </div>
      </button>

      <button
        onClick={() => onChange('items')}
        className={`flex flex-col items-center gap-1 transition active:scale-95 ${active === 'items' ? 'text-nesio-accent' : 'text-nesio-muted'}`}
      >
        <IconBox className="w-6 h-6" />
        <span className="type-caption">物品</span>
      </button>
    </nav>
  )
}
