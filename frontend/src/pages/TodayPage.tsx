import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  IconCloud, IconMic, IconPlus, IconX
} from '../icons'
import { today } from '../api/client'

interface TodayCard {
  id: string
  title: string
  body?: string
  severity: number
  slot: string
}

interface TodayResponse {
  cards: TodayCard[]
  local_day: string
}

interface Props {
  onMemory?: () => void
  onSettings?: () => void
  onChat?: () => void
}

export default function TodayPage({ onMemory, onSettings, onChat }: Props) {
  // avoid unused parameter errors when handlers are not passed
  void onMemory
  void onSettings
  void onChat
  const [feeling, setFeeling] = useState('')
  const { data, isLoading } = useQuery<TodayResponse>({
    queryKey: ['today-cards'],
    queryFn: async () => {
      const response = await today.get()
      return response.data as TodayResponse
    },
  })

  const cards = data?.cards ?? []
  const primaryCard = cards[0]

  return (
    <div className="px-5 pt-6 pb-4 space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        {/* 左上角 Logo - 点击进入记忆/洞察页 */}
        <button
          onClick={onMemory}
          className="w-10 h-10 rounded-xl bg-nesio-accentSoft flex items-center justify-center active:scale-95 transition"
        >
          <div className="w-6 h-6 rounded bg-nesio-accent opacity-60" />
        </button>

        {/* 天气 */}
        <div className="flex items-center gap-2 bg-white rounded-full px-3 py-1.5 shadow-card">
          <IconCloud className="w-4 h-4 text-nesio-muted" />
          <span className="text-sm text-nesio-ink">26°</span>
        </div>

        {/* 右上角头像 - 点击进入设置页 */}
        <button
          onClick={onSettings}
          className="w-10 h-10 rounded-full bg-nesio-accentSoft overflow-hidden active:scale-95 transition"
        >
          <div className="w-full h-full bg-nesio-accentLight flex items-center justify-center">
            <span className="text-sm font-bold text-nesio-accent">婧</span>
          </div>
        </button>
      </div>

      {/* Greeting */}
      <div>
        <h1 className="text-[26px] font-bold leading-snug text-nesio-ink">
          婧,晚上好。今天有 {cards.length} 条待处理内容，先看最重要的一条。
        </h1>
      </div>

      {/* Today Section Card */}
      <div className="nesio-card p-4 flex items-center gap-3 active:scale-[0.99] transition">
        <div className="w-10 h-10 rounded-xl bg-nesio-icon-bg flex items-center justify-center shrink-0">
          <svg className="w-5 h-5 text-nesio-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
            <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
            <polyline points="14 2 14 8 20 8"/>
          </svg>
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-base font-medium text-nesio-ink">{primaryCard?.title ?? '今天这一段'}</div>
          <div className="text-sm text-nesio-muted truncate">
            {isLoading ? '正在同步今天卡片...' : primaryCard?.body ?? `本地日历日 ${data?.local_day ?? '--'}`}
          </div>
        </div>
        <svg className="w-5 h-5 text-nesio-muted shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M9 18l6-6-6-6"/>
        </svg>
      </div>

      {/* Input */}
      <div className="flex items-center gap-3">
        <button className="w-10 h-10 rounded-full bg-white shadow-card flex items-center justify-center text-nesio-muted active:scale-95 transition">
          <IconPlus className="w-5 h-5" />
        </button>
        <div className="flex-1 bg-white rounded-full px-4 py-2.5 shadow-card flex items-center">
          <input
            type="text"
            placeholder=""
            className="flex-1 bg-transparent outline-none text-sm"
          />
        </div>
        <button className="w-10 h-10 rounded-full bg-nesio-accent flex items-center justify-center text-white active:scale-95 transition">
          <IconMic className="w-5 h-5" />
        </button>
      </div>

      {/* Timeline */}
      <div className="space-y-4">
        {/* Now */}
        <div className="flex gap-3">
          <div className="flex flex-col items-center gap-1">
            <div className="w-8 h-8 rounded-full bg-nesio-accentSoft flex items-center justify-center">
              <svg className="w-4 h-4 text-nesio-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
              </svg>
            </div>
            <div className="w-0.5 flex-1 bg-nesio-border rounded-full" />
          </div>
          <div className="flex-1 pb-4">
            <div className="text-sm text-nesio-accent font-medium">现在</div>
            <div className="text-lg font-medium text-nesio-ink mt-0.5">此刻,你感觉——</div>
            <div className="mt-2 flex gap-2 flex-wrap">
              {['😊 平静', '😰 焦虑', '😴 疲惫', '💪 充满能量'].map((m) => (
                <button
                  key={m}
                  onClick={() => setFeeling(m)}
                  className={`px-3 py-1.5 rounded-full text-sm border transition ${
                    feeling === m
                      ? 'bg-nesio-accent text-white border-nesio-accent'
                      : 'bg-white text-nesio-ink border-nesio-border'
                  }`}
                >
                  {m}
                </button>
              ))}
            </div>
          </div>
        </div>

        {cards.slice(1, 4).map((card) => (
          <div key={card.id} className="flex gap-3">
            <div className="flex flex-col items-center gap-1">
              <div className="w-8 h-8 rounded-full border-2 border-nesio-accent flex items-center justify-center">
                <svg className="w-4 h-4 text-nesio-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>
                </svg>
              </div>
            </div>
            <div className="flex-1 flex items-start justify-between gap-3">
              <div>
                <div className="text-sm text-nesio-accent font-medium">{card.slot} · 严重度 {card.severity}</div>
                <div className="text-lg font-medium text-nesio-ink mt-0.5">{card.title}</div>
                {card.body && <div className="text-sm text-nesio-muted mt-1">{card.body}</div>}
              </div>
              <button className="w-6 h-6 rounded-full bg-nesio-icon-bg flex items-center justify-center text-nesio-muted active:scale-90 transition">
                <IconX className="w-3 h-3" />
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
