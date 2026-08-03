import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  IconSearch, IconBookmark, IconBox, IconFolder,
  IconUser, IconFileText, IconCalendar, IconArrowLeft
} from '../icons'
import { memories } from '../api/client'

const sources = [
  { name: '笔记', count: 0, icon: IconFileText },
  { name: '事件', count: 0, icon: IconCalendar },
]

interface Props {
  onBack: () => void
  initialDomainHint?: string | null
}

interface MemoryItem {
  id: string
  type: string
  domain?: string
  title: string
  body?: string
  tags?: string[] | null
  created_at: string
}

function sanitizeBodyForDisplay(body?: string): string {
  if (!body) {
    return ''
  }
  if (/AI not configured\. Add GEMINI_API_KEY/i.test(body)) {
    return ''
  }
  return body
}

export default function MemoryPage({ onBack, initialDomainHint }: Props) {
  const [activeTag, setActiveTag] = useState('全部')
  const [query, setQuery] = useState('')
  const [selectedItem, setSelectedItem] = useState<MemoryItem | null>(null)

  useEffect(() => {
    if (initialDomainHint) {
      setQuery(initialDomainHint)
    }
  }, [initialDomainHint])
  const { data } = useQuery({
    queryKey: ['memories'],
    queryFn: async () => {
      const response = await memories.list()
      const rows = (response.data ?? []) as MemoryItem[]
      return rows.map((item) => ({
        ...item,
        tags: Array.isArray(item.tags) ? item.tags : [],
      }))
    },
  })

  const allItems = data ?? []
  const categories = [
    { name: '全部', count: allItems.length },
    { name: '任务', count: allItems.filter((item) => item.type === 'task').length },
    { name: '记忆', count: allItems.filter((item) => item.type === 'memory').length },
    { name: '其他', count: allItems.filter((item) => !['task', 'memory'].includes(item.type)).length },
  ]
  const memoryItems = allItems.filter((item) => {
    const tags = Array.isArray(item.tags) ? item.tags : []
    const matchesCategory = activeTag === '全部'
      || (activeTag === '任务' && item.type === 'task')
      || (activeTag === '记忆' && item.type === 'memory')
      || (activeTag === '其他' && !['task', 'memory'].includes(item.type))
    const searchText = `${item.title} ${sanitizeBodyForDisplay(item.body)} ${item.domain ?? ''} ${tags.join(' ')}`.toLowerCase()
    return matchesCategory && searchText.includes(query.trim().toLowerCase())
  })

  if (selectedItem) {
    const tags = Array.isArray(selectedItem.tags) ? selectedItem.tags : []
    return (
      <div className="px-5 pt-6 pb-4 space-y-5">
        <div className="flex items-center gap-3">
          <button
            onClick={() => setSelectedItem(null)}
            className="ui-icon-btn w-8 h-8"
          >
            <IconArrowLeft className="w-5 h-5" />
          </button>
          <h2 className="type-h2 text-nesio-ink">记忆详情</h2>
        </div>

        <article className="ui-card p-5 space-y-4">
          <div className="flex items-start justify-between gap-3">
            <h3 className="type-title text-nesio-ink break-words">{selectedItem.title}</h3>
            <span className="shrink-0 rounded-full bg-nesio-accentSoft px-2 py-1 type-caption text-nesio-accent">
              {selectedItem.type === 'task' ? '任务' : selectedItem.type === 'memory' ? '记忆' : selectedItem.type}
            </span>
          </div>

          <div className="type-caption text-nesio-muted">
            创建时间 {new Date(selectedItem.created_at).toLocaleString('zh-CN', { hour12: false })}
          </div>

          {sanitizeBodyForDisplay(selectedItem.body) && (
            <div className="type-body text-nesio-ink whitespace-pre-wrap break-words">{sanitizeBodyForDisplay(selectedItem.body)}</div>
          )}

          {(selectedItem.domain || tags.length > 0) && (
            <div className="flex flex-wrap gap-2">
              {selectedItem.domain && (
                <span className="px-2 py-1 rounded-full bg-nesio-bg type-caption text-nesio-muted">
                  {selectedItem.domain}
                </span>
              )}
              {tags.map((tag) => (
                <span key={tag} className="px-2 py-1 rounded-full bg-nesio-bg type-caption text-nesio-muted">
                  {tag}
                </span>
              ))}
            </div>
          )}
        </article>
      </div>
    )
  }

  return (
    <div className="px-5 pt-6 pb-4 space-y-5">
      {/* Header with back */}
      <div className="flex items-center gap-3">
        <button
          onClick={onBack}
          className="ui-icon-btn w-8 h-8"
        >
          <IconArrowLeft className="w-5 h-5" />
        </button>
        <h2 className="type-h2 text-nesio-ink">搜记忆</h2>
      </div>

      {/* Search */}
      <div className="relative">
        <IconSearch className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-nesio-muted" />
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜记忆"
          className="ui-input pl-10 shadow-card"
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
        <div className="type-title text-nesio-ink mb-3">
          全部记忆映射 · {allItems.length} 条
        </div>

        {/* Tags */}
        <div className="flex flex-wrap gap-2 mb-4">
          {categories.map((t) => (
            <button
              key={t.name}
              onClick={() => setActiveTag(t.name)}
              className={`ui-chip transition ${
                activeTag === t.name
                  ? 'ui-chip-active'
                  : ''
              }`}
            >
              {t.name === '任务' && <IconCalendar className="w-3.5 h-3.5" />}
              {t.name === '记忆' && <IconFileText className="w-3.5 h-3.5" />}
              {t.name === '其他' && <IconUser className="w-3.5 h-3.5" />}
              <span>{t.name}</span>
              {t.count > 0 && (
                <span className={`type-caption px-1.5 py-0.5 rounded-full ${activeTag === t.name ? 'bg-white/20' : 'bg-nesio-bg'}`}>
                  {t.count}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* Expandable Section */}
        <button className="w-full ui-card-plain px-4 py-3.5 flex items-center justify-between mb-3 active:scale-[0.99] transition">
          <span className="type-body text-nesio-ink">交易 · 1440 笔</span>
          <span className="type-caption text-nesio-muted">—— 展开</span>
        </button>

        {/* Source Cards */}
        <div className="space-y-3">
          {memoryItems.slice(0, 8).map((item) => {
            const tags = Array.isArray(item.tags) ? item.tags : []
            return (
              <button
                key={item.id}
                type="button"
                onClick={() => setSelectedItem(item)}
                className="ui-card-plain p-4 w-full text-left active:scale-[0.99] transition"
              >
              <div className="flex items-start justify-between gap-3">
                <div className="type-body font-medium text-nesio-ink">{item.title}</div>
                <span className="shrink-0 rounded-full bg-nesio-accentSoft px-2 py-1 type-caption text-nesio-accent">
                  {item.type === 'task' ? '任务' : item.type === 'memory' ? '记忆' : item.type}
                </span>
              </div>
              {sanitizeBodyForDisplay(item.body) && <div className="type-body text-nesio-muted mt-1">{sanitizeBodyForDisplay(item.body)}</div>}
              {(item.domain || tags.length > 0) && (
                <div className="flex flex-wrap gap-2 mt-3">
                  {item.domain && (
                    <span className="px-2 py-1 rounded-full bg-nesio-bg type-caption text-nesio-muted">
                      {item.domain}
                    </span>
                  )}
                  {tags.map((tag) => (
                    <span key={tag} className="px-2 py-1 rounded-full bg-nesio-bg type-caption text-nesio-muted">
                      {tag}
                    </span>
                  ))}
                </div>
              )}
              </button>
            )
          })}
          {memoryItems.length === 0 && (
            <div className="type-body text-nesio-muted">当前筛选下还没有云端记忆映射。</div>
          )}
          {memoryItems.length > 0 && (
            <div className="grid grid-cols-2 gap-3">
              {sources.map((s) => (
                <button key={s.name} className="ui-card-plain p-4 flex items-center gap-3 active:scale-[0.98] transition">
                  <div className="w-10 h-10 rounded-xl bg-nesio-accentSoft flex items-center justify-center">
                    <s.icon className="w-5 h-5 text-nesio-accent" />
                  </div>
                  <span className="type-body font-medium text-nesio-ink">{s.name}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
