import { useState, useRef, useEffect } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { IconArrowLeft, IconClock, IconMic, IconPlus } from '../icons'
import { chat, intake } from '../api/client'

interface Props {
  onBack: () => void
}

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  memories?: { icon: string; title: string }[]
}

interface SpeechRecognitionResultLike {
  transcript?: string
}

interface SpeechRecognitionEventLike {
  results: ArrayLike<ArrayLike<SpeechRecognitionResultLike>>
}

interface SpeechRecognitionLike {
  lang: string
  interimResults: boolean
  maxAlternatives: number
  continuous?: boolean
  onresult: ((event: SpeechRecognitionEventLike) => void) | null
  onerror: ((event: { error?: string }) => void) | null
  onend: (() => void) | null
  start: () => void
  stop: () => void
}

interface SpeechRecognitionConstructorLike {
  new (): SpeechRecognitionLike
}

interface WindowWithSpeechRecognition extends Window {
  SpeechRecognition?: SpeechRecognitionConstructorLike
  webkitSpeechRecognition?: SpeechRecognitionConstructorLike
}

export default function ChatPage({ onBack }: Props) {
  const [messages, setMessages] = useState<Message[]>([])
  const [draft, setDraft] = useState('')
  const [isListening, setIsListening] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const recognitionRef = useRef<SpeechRecognitionLike | null>(null)

  const historyQuery = useQuery({
    queryKey: ['chat-history'],
    queryFn: async () => {
      const response = await chat.history()
      return response.data as Array<{ id: string; role: 'user' | 'assistant'; content: string }>
    },
  })

  useEffect(() => {
    if (historyQuery.data) {
      setMessages([...historyQuery.data].reverse())
    }
  }, [historyQuery.data])

  const sendMutation = useMutation({
    mutationFn: async (message: string) => {
      const response = await chat.send(message)
      return response.data as { content: string }
    },
    onSuccess: (result, message) => {
      setMessages((current) => [
        ...current,
        { id: `user-${Date.now()}`, role: 'user', content: message },
        { id: `assistant-${Date.now()}`, role: 'assistant', content: result.content },
      ])
      setDraft('')
    },
    onError: (_error, message) => {
      setMessages((current) => [
        ...current,
        { id: `user-${Date.now()}`, role: 'user', content: message },
        { id: `assistant-${Date.now()}`, role: 'assistant', content: '当前离线，这条消息已加入发送队列，网络恢复后会自动重试。' },
      ])
      setDraft('')
    },
  })

  const uploadMutation = useMutation({
    mutationFn: async (file: File) => {
      await intake.upload(file)
      return file.name
    },
    onSuccess: (filename) => {
      setMessages((current) => [
        ...current,
        { id: `user-upload-${Date.now()}`, role: 'user', content: `已上传文件：${filename}` },
        { id: `assistant-upload-${Date.now()}`, role: 'assistant', content: '文件已上传并开始识别，稍后可在物品/记忆中查看结果。' },
      ])
    },
    onError: () => {
      setMessages((current) => [
        ...current,
        { id: `assistant-upload-error-${Date.now()}`, role: 'assistant', content: '上传失败，请检查网络后重试。' },
      ])
    },
  })

  const bottomRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  useEffect(() => {
    return () => {
      recognitionRef.current?.stop()
      recognitionRef.current = null
    }
  }, [])

  const submit = () => {
    if (!draft.trim() || sendMutation.isPending) {
      return
    }
    sendMutation.mutate(draft.trim())
  }

  const toggleVoiceInput = () => {
    const Constructor = (window as WindowWithSpeechRecognition).SpeechRecognition
      ?? (window as WindowWithSpeechRecognition).webkitSpeechRecognition
    if (!Constructor) {
      setMessages((current) => [
        ...current,
        { id: `assistant-no-speech-${Date.now()}`, role: 'assistant', content: '当前设备不支持语音输入。' },
      ])
      return
    }

    if (isListening) {
      recognitionRef.current?.stop()
      setIsListening(false)
      return
    }

    const recognition = new Constructor()
    recognition.lang = 'zh-CN'
    recognition.continuous = false
    recognition.interimResults = false
    recognition.maxAlternatives = 1
    recognition.onresult = (event) => {
      const transcript = event.results[0]?.[0]?.transcript?.trim() ?? ''
      if (!transcript) {
        return
      }
      setDraft((current) => (current ? `${current} ${transcript}` : transcript))
    }
    recognition.onerror = (event) => {
      const reason = event?.error ? `（${event.error}）` : ''
      setMessages((current) => [
        ...current,
        { id: `assistant-speech-error-${Date.now()}`, role: 'assistant', content: `语音输入失败，请检查麦克风权限${reason}` },
      ])
      setIsListening(false)
      recognitionRef.current = null
    }
    recognition.onend = () => {
      setIsListening(false)
      recognitionRef.current = null
    }

    recognitionRef.current = recognition
    try {
      recognition.start()
      setIsListening(true)
    } catch {
      setMessages((current) => [
        ...current,
        { id: `assistant-speech-start-error-${Date.now()}`, role: 'assistant', content: '无法启动语音输入，请检查浏览器与麦克风权限。' },
      ])
      setIsListening(false)
      recognitionRef.current = null
    }
  }

  const openUpload = () => {
    fileInputRef.current?.click()
  }

  return (
    <div className="h-full flex flex-col bg-nesio-bg">
      {/* Header */}
      <div className="px-4 pt-4 pb-2 flex items-center justify-between shrink-0">
        <button onClick={onBack} className="ui-icon-btn text-nesio-accent" aria-label="返回">
          <IconArrowLeft className="w-6 h-6" />
        </button>
        <h2 className="type-h2 text-nesio-ink">念念</h2>
        <div className="flex items-center gap-3">
          <button className="ui-icon-btn text-nesio-muted" aria-label="历史">
            <IconClock className="w-5 h-5" />
          </button>
          <button className="ui-btn-secondary px-3">新对话</button>
        </div>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-4">
        {historyQuery.isLoading && messages.length === 0 && (
          <div className="ui-state-info">正在加载对话...</div>
        )}
        {messages.map((msg) => (
          <div key={msg.id} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
            <div className={`max-w-[85%] ${msg.role === 'user' ? 'bg-nesio-accent text-white rounded-2xl rounded-tr-sm px-4 py-2.5' : 'space-y-3'}`}>
              {msg.role === 'assistant' && (
                <>
                  <div className="bg-white rounded-2xl rounded-tl-sm px-4 py-3 shadow-card text-nesio-ink type-body leading-relaxed whitespace-pre-line">
                    {msg.content}
                  </div>
                  {msg.memories && (
                    <div>
                      <div className="type-caption text-nesio-muted mb-2">相关记忆 · 点开可回看/回复</div>
                      <div className="space-y-2">
                        {msg.memories.map((m, i) => (
                          <button key={i} className="w-full ui-btn-ghost justify-start gap-2.5 px-3">
                            <span className="w-2.5 h-2.5 rounded-full bg-nesio-accentLight shrink-0" />
                            <span className="type-body text-nesio-ink truncate">{m.title}</span>
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </>
              )}
              {msg.role === 'user' && (
                <span className="type-body">{msg.content}</span>
              )}
            </div>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <div className="px-4 pt-2 pb-4 shrink-0 tabbar-safe">
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*,.pdf,.txt,.md"
          className="hidden"
          onChange={(event) => {
            const file = event.currentTarget.files?.[0]
            if (file) {
              uploadMutation.mutate(file)
            }
            event.currentTarget.value = ''
          }}
        />
        <div className="flex items-center gap-2">
          <button onClick={toggleVoiceInput} className={`ui-icon-btn ${isListening ? 'bg-nesio-accent text-white' : ''}`} aria-label="语音输入">
            <IconMic className="w-5 h-5" />
          </button>
          <div className="flex-1">
            <input
              type="text"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  submit()
                }
              }}
              placeholder="问一问..."
              className="ui-input-pill"
            />
          </div>
          <button onClick={openUpload} className="ui-icon-btn" aria-label="上传文件">
            <IconPlus className="w-5 h-5" />
          </button>
          <button onClick={submit} className="ui-btn-primary px-3" aria-label="发送">发送</button>
        </div>
      </div>
    </div>
  )
}
