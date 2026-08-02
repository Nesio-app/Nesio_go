import { useState, useRef, useEffect } from 'react'
import { IconArrowLeft, IconClock, IconMic, IconPlus } from '../icons'

interface Props {
  onBack: () => void
}

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  memories?: { icon: string; title: string }[]
}

export default function ChatPage({ onBack }: Props) {
  const [messages] = useState<Message[]>([
    {
      id: '1',
      role: 'assistant',
      content: `1. NC Leap Alumni Ice Cream Social with Fidterns! (在 Creekside)
2. Wash Dish
3. FAB Sprint Planning
4. Go/No-Go PROD | ADE.2...`,
    },
    {
      id: '2',
      role: 'user',
      content: '你也',
    },
    {
      id: '3',
      role: 'assistant',
      content: '我记得你之前提到过，你觉得"着急是一种非常大的负能量，越是急性...',
      memories: [
        { icon: '📅', title: 'Wash Dish' },
        { icon: '📅', title: 'FAB Sprint Planning' },
        { icon: '📅', title: 'Go/No-Go PROD | ADE.2...' },
      ],
    },
  ])

  const bottomRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  return (
    <div className="h-full flex flex-col bg-nesio-bg">
      {/* Header */}
      <div className="px-4 pt-4 pb-2 flex items-center justify-between shrink-0">
        <button onClick={onBack} className="w-8 h-8 flex items-center justify-center text-nesio-accent active:scale-95 transition">
          <IconArrowLeft className="w-6 h-6" />
        </button>
        <h2 className="text-lg font-bold text-nesio-ink">念念</h2>
        <div className="flex items-center gap-3">
          <button className="w-8 h-8 flex items-center justify-center text-nesio-muted">
            <IconClock className="w-5 h-5" />
          </button>
          <button className="text-sm text-nesio-muted">新对话</button>
        </div>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-4">
        {messages.map((msg) => (
          <div key={msg.id} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
            <div className={`max-w-[85%] ${msg.role === 'user' ? 'bg-nesio-accent text-white rounded-2xl rounded-tr-sm px-4 py-2.5' : 'space-y-3'}`}>
              {msg.role === 'assistant' && (
                <>
                  <div className="bg-white rounded-2xl rounded-tl-sm px-4 py-3 shadow-card text-nesio-ink text-[15px] leading-relaxed whitespace-pre-line">
                    {msg.content}
                  </div>
                  {msg.memories && (
                    <div>
                      <div className="text-xs text-nesio-muted mb-2">相关记忆 · 点开可回看/回复</div>
                      <div className="space-y-2">
                        {msg.memories.map((m, i) => (
                          <button key={i} className="w-full bg-white rounded-xl px-3 py-2.5 flex items-center gap-2.5 shadow-card active:scale-[0.98] transition">
                            <span className="text-base">{m.icon}</span>
                            <span className="text-sm text-nesio-ink">{m.title}</span>
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </>
              )}
              {msg.role === 'user' && (
                <span className="text-[15px]">{msg.content}</span>
              )}
            </div>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <div className="px-4 pb-6 pt-2 shrink-0">
        <div className="flex items-center gap-2">
          <button className="w-10 h-10 rounded-full bg-white shadow-card flex items-center justify-center text-nesio-muted active:scale-95 transition">
            <IconMic className="w-5 h-5" />
          </button>
          <div className="flex-1 bg-white rounded-full px-4 py-2.5 shadow-card flex items-center">
            <input
              type="text"
              placeholder="问一问..."
              className="flex-1 bg-transparent outline-none text-sm text-nesio-ink placeholder:text-nesio-muted"
            />
          </div>
          <button className="w-10 h-10 rounded-full bg-white shadow-card flex items-center justify-center text-nesio-muted active:scale-95 transition">
            <IconPlus className="w-5 h-5" />
          </button>
        </div>
      </div>
    </div>
  )
}
