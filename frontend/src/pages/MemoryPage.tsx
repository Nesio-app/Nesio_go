import { useState } from 'react'
import {
  IconSearch, IconBookmark, IconBox, IconFolder,
  IconUser, IconFileText, IconCalendar, IconArrowLeft
} from '../icons'

const tags = [
  { name: '全部', count: 3318, active: true },
  { name: '人物', count: 133, active: false },
  { name: '物品', count: 0, active: false },
  { name: '手记', count: 1333, active: false },
  { name: '系统', count: 0, active: false },
  { name: '财务', count: 1441, active: false },
  { name: '健康', count: 1, active: false },
]

const sources = [
  { name: '笔记', count: 0, icon: IconFileText },
  { name: '事件', count: 0, icon: IconCalendar },
]

interface Props {
  onBack: () => void
}

export default function MemoryPage({ onBack }: Props) {
  const [activeTag, setActiveTag] = useState('全部')
  const [query, setQuery] = useState('')

  return (
    <div className="px-5 pt-6 pb-4 space-y-5">
      {/* Header with back */}
      <div className="flex items-center gap-3">
        <button
          onClick={onBack}
          className="w-8 h-8 flex items-center justify-center text-nesio-ink active:scale-90 transition"
        >
          <IconArrowLeft className="w-5 h-5" />
        </button>
        <h2 className="text-lg font-bold text-nesio-ink">搜记忆</h2>
      </div>

      {/* Search */}
      <div className="relative">
        <IconSearch className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-nesio-muted" />
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜记忆"
          className="w-full bg-white rounded-2xl pl-10 pr-4 py-3 text-base outline-none shadow-card placeholder:text-nesio-muted"
        />
      </div>

      {/* Top Categories */}
      <div className="flex gap-4">
        {[
          { icon: IconBookmark, label: '收藏夹', sub: '手动收 · 1' },
          { icon: IconBox, label: '收纳', sub: '10 件' },
          { icon: IconFolder, label: '项目', sub: '进行 · 1' },
        ].map((item) => (
          <button key={item.label} className="flex-1 flex flex-col items-center gap-2 active:scale-95 transition">
            <div className="w-14 h-14 rounded-full bg-white shadow-card flex items-center justify-center">
              <item.icon className="w-6 h-6 text-nesio-accent" />
            </div>
            <div className="text-sm font-medium text-nesio-ink">{item.label}</div>
            <div className="text-xs text-nesio-muted">{item.sub}</div>
          </button>
        ))}
      </div>

      {/* All Memories */}
      <div>
        <div className="text-base font-bold text-nesio-ink mb-3">
          全部记忆 · 3318 条 · 可搜
        </div>

        {/* Tags */}
        <div className="flex flex-wrap gap-2 mb-4">
          {tags.map((t) => (
            <button
              key={t.name}
              onClick={() => setActiveTag(t.name)}
              className={`px-3 py-1.5 rounded-full text-sm flex items-center gap-1.5 transition ${
                activeTag === t.name
                  ? 'bg-nesio-accent text-white'
                  : 'bg-white text-nesio-ink border border-nesio-border'
              }`}
            >
              {t.name === '人物' && <IconUser className="w-3.5 h-3.5" />}
              {t.name === '物品' && <IconBox className="w-3.5 h-3.5" />}
              {t.name === '手记' && <IconFileText className="w-3.5 h-3.5" />}
              {t.name === '系统' && <IconCalendar className="w-3.5 h-3.5" />}
              <span>{t.name}</span>
              {t.count > 0 && (
                <span className={`text-xs px-1.5 py-0.5 rounded-full ${activeTag === t.name ? 'bg-white/20' : 'bg-nesio-bg'}`}>
                  {t.count}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* Expandable Section */}
        <button className="w-full bg-white rounded-2xl px-4 py-3.5 flex items-center justify-between shadow-card mb-3 active:scale-[0.99] transition">
          <span className="text-base text-nesio-ink">交易 · 1440 笔</span>
          <span className="text-sm text-nesio-muted">—— 展开</span>
        </button>

        {/* Source Cards */}
        <div className="grid grid-cols-2 gap-3">
          {sources.map((s) => (
            <button key={s.name} className="bg-white rounded-2xl p-4 flex items-center gap-3 shadow-card active:scale-[0.98] transition">
              <div className="w-10 h-10 rounded-xl bg-nesio-accentSoft flex items-center justify-center">
                <s.icon className="w-5 h-5 text-nesio-accent" />
              </div>
              <span className="text-base font-medium text-nesio-ink">{s.name}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
