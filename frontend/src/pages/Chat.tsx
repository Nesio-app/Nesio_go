import { useState, useRef, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { chat } from '../api/client'

export default function ChatPage() {
  const [input, setInput] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)
  const qc = useQueryClient()
  const nav = useNavigate()

  const { data: history } = useQuery({ queryKey: ['chat-history'], queryFn: () => chat.history().then(r => r.data) })
  const send = useMutation({
    mutationFn: (message: string) => chat.send(message),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chat-history'] }),
  })

  const messages = history || []
  useEffect(() => { bottomRef.current?.scrollIntoView({ behavior: 'smooth' }) }, [messages])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim() || send.isPending) return
    send.mutate(input.trim())
    setInput('')
  }

  return (
    <div className="min-h-screen flex flex-col">
      <div className="flex items-center justify-between px-5 pt-6 pb-3">
        <button onClick={() => nav(-1)} className="text-white">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><polyline points="15 18 9 12 15 6"/></svg>
        </button>
        <h1 className="text-white font-bold text-lg drop-shadow">NianNian</h1>
        <button onClick={() => nav('/')} className="avatar">J</button>
      </div>

      <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
        {messages.length === 0 && <div className="text-center text-white/70 py-10">Hello, I am NianNian</div>}
        {messages.slice().reverse().map((msg: any) => (
          <div key={msg.id} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
            {msg.role === 'assistant' && (
              <div className="w-8 h-8 rounded-xl bg-white/90 flex items-center justify-center mr-2 flex-shrink-0">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="#6B9FD4"><path d="M12 2L2 7l10 5 10-5-10-5z"/></svg>
              </div>
            )}
            <div className={`max-w-[80%] px-4 py-3 rounded-2xl text-[15px] leading-relaxed ${msg.role === 'user' ? 'bg-[#6B9FD4] text-white rounded-br-md' : 'bg-white text-[#1E293B] rounded-bl-md shadow-sm'}`}>
              {msg.content}
            </div>
          </div>
        ))}
        {send.isPending && (
          <div className="flex justify-start">
            <div className="bg-white rounded-2xl rounded-bl-md px-4 py-3 shadow-sm">
              <span className="animate-pulse text-[#94A3B8]">Thinking...</span>
            </div>
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      <form onSubmit={handleSubmit} className="px-4 pb-8 pt-2">
        <div className="bg-white rounded-full shadow-lg flex items-center gap-2 px-4 py-2">
          <button type="button" className="w-8 h-8 rounded-full bg-[#F0F4F8] flex items-center justify-center text-[#94A3B8]">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/></svg>
          </button>
          <input type="text" value={input} onChange={(e) => setInput(e.target.value)} placeholder="Ask something..." className="flex-1 bg-transparent text-[15px] outline-none" />
          <button type="submit" disabled={send.isPending} className="w-8 h-8 rounded-full bg-[#6B9FD4] flex items-center justify-center text-white">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
          </button>
        </div>
      </form>
    </div>
  )
}