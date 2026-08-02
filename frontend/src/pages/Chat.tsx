import { useState, useRef, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { chat } from '../api/client'

export default function ChatPage() {
  const [input, setInput] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)
  const qc = useQueryClient()

  const { data: history, isLoading } = useQuery({
    queryKey: ['chat-history'],
    queryFn: () => chat.history().then((r) => r.data),
  })

  const send = useMutation({
    mutationFn: (message: string) => chat.send(message),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chat-history'] }),
  })

  const messages = history || []

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim() || send.isPending) return
    send.mutate(input.trim())
    setInput('')
  }

  return (
    <div className="flex flex-col h-[calc(100vh-120px)]">
      <h2 className="text-xl font-bold mb-4">念念</h2>
      <div className="flex-1 overflow-y-auto space-y-4 pb-4">
        {messages.length === 0 && !isLoading && (
          <div className="text-center text-[var(--color-text-secondary)] py-10">
            你好，我是念念。有什么可以帮你的？
          </div>
        )}
        {messages.slice().reverse().map((msg: any) => (
          <div
            key={msg.id}
            className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
          >
            <div
              className={`max-w-[80%] px-4 py-3 rounded-2xl text-sm ${
                msg.role === 'user'
                  ? 'bg-[var(--color-portal-blue)] text-white rounded-br-md'
                  : 'bg-white border border-[var(--color-border)] rounded-bl-md'
              }`}
            >
              {msg.content}
            </div>
          </div>
        ))}
        {send.isPending && (
          <div className="flex justify-start">
            <div className="bg-white border border-[var(--color-border)] px-4 py-3 rounded-2xl rounded-bl-md">
              <span className="animate-pulse text-[var(--color-text-secondary)]">思考中...</span>
            </div>
          </div>
        )}
        <div ref={bottomRef} />
      </div>
      <form onSubmit={handleSubmit} className="flex gap-2 pt-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="输入消息..."
          className="flex-1 px-4 py-3 rounded-xl border border-[var(--color-border)] bg-white focus:outline-none focus:ring-2 focus:ring-[var(--color-portal-blue)]"
        />
        <button type="submit" disabled={send.isPending} className="btn-primary px-6">
          发送
        </button>
      </form>
    </div>
  )
}
