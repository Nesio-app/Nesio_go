import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  IconCloud, IconMic, IconPlus, IconX
} from '../icons'
import { dailyBriefs, extraction, intake, mention, reminders, search, today } from '../api/client'

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
  onChat?: (options?: { draft?: string; autoSubmit?: boolean }) => void
}

interface IntakeIngestResponse {
  node_id: string
  reminder_created: boolean
  intent: string
  intent_label: string
  confidence: number
  remind_at?: string | null
}

interface MentionResult {
  id: string
  type: string
  title: string
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

function formatIntakeHint(data: IntakeIngestResponse): string {
  const confidence = Number.isFinite(data.confidence)
    ? `${Math.round(Math.max(0, Math.min(1, data.confidence)) * 100)}%`
    : '--'

  if (data.reminder_created && data.remind_at) {
    const remindDate = new Date(data.remind_at)
    if (!Number.isNaN(remindDate.getTime())) {
      const timeText = new Intl.DateTimeFormat('zh-CN', {
        month: 'numeric',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      }).format(remindDate)
      return `识别为${data.intent_label}，将在 ${timeText} 提醒（置信度 ${confidence}）。`
    }
  }

  return `识别为${data.intent_label}并已入库（置信度 ${confidence}）。`
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

function inferReminderAt(text: string): Date | null {
  const source = text.trim()
  if (!source) {
    return null
  }
  const now = new Date()
  const reminder = new Date(now)

  if (source.includes('后天')) {
    reminder.setDate(reminder.getDate() + 2)
  } else if (source.includes('明天')) {
    reminder.setDate(reminder.getDate() + 1)
  } else if (!source.includes('今天') && !source.match(/\d{1,2}\s*点/)) {
    return null
  }

  const hourMatch = source.match(/(\d{1,2})\s*点/)
  if (hourMatch) {
    const hour = Number(hourMatch[1])
    if (hour >= 0 && hour <= 23) {
      reminder.setHours(hour, 0, 0, 0)
      return reminder
    }
  }

  reminder.setHours(now.getHours() + 2, 0, 0, 0)
  return reminder
}

export default function TodayPage({ onMemory, onSettings, onChat }: Props) {
  // avoid unused parameter errors when handlers are not passed
  void onMemory
  void onSettings
  void onChat
  const [feeling, setFeeling] = useState('')
  const [taskTitle, setTaskTitle] = useState('')
  const [saveMessage, setSaveMessage] = useState('')
  const [saveTone, setSaveTone] = useState<'info' | 'success' | 'error'>('info')
  const [intakeHint, setIntakeHint] = useState('')
  const [searchResults, setSearchResults] = useState<Array<{ id: string; title: string; type: string }>>([])
  const [mentionResults, setMentionResults] = useState<MentionResult[]>([])
  const [isListening, setIsListening] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const recognitionRef = useRef<SpeechRecognitionLike | null>(null)
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery<TodayResponse>({
    queryKey: ['today-cards'],
    queryFn: async () => {
      const response = await today.get()
      return response.data as TodayResponse
    },
  })

  const cards = data?.cards ?? []
  const primaryCard = cards[0]
  const briefQuery = useQuery({
    queryKey: ['daily-brief-today'],
    queryFn: async () => {
      const response = await dailyBriefs.today()
      return response.data as { content: string }
    },
  })

  const intakeMutation = useMutation({
    mutationFn: async (text: string) => {
      const response = await intake.ingest(text)
      return response.data as IntakeIngestResponse
    },
    onSuccess: (data) => {
      setTaskTitle('')
      setSearchResults([])
      setSaveMessage('已识别并存储。')
      setSaveTone('success')
      setIntakeHint(formatIntakeHint(data))
      void Promise.all([
        queryClient.invalidateQueries({ queryKey: ['today-cards'] }),
        queryClient.invalidateQueries({ queryKey: ['memories'] }),
      ])
    },
    onError: () => {
      setSaveMessage('识别失败，请稍后重试。')
      setSaveTone('error')
      setIntakeHint('')
    },
  })

  const searchMutation = useMutation({
    mutationFn: async (keyword: string) => {
      const response = await search.query(keyword)
      return response.data as Array<{ id: string; title: string; type: string }>
    },
    onSuccess: (rows) => {
      setSearchResults(rows)
      setSaveMessage(rows.length > 0 ? `找到 ${rows.length} 条相关记录` : '没有找到相关记录')
      setSaveTone('info')
    },
    onError: () => {
      setSaveMessage('搜索失败，请稍后重试。')
      setSaveTone('error')
    },
  })

  const uploadMutation = useMutation({
    mutationFn: async ({ files, note }: { files: File[]; note?: string }) => {
      await extraction.upload(files, note)
    },
    onSuccess: async () => {
      setSaveMessage('文件已智能识别并入库。')
      setSaveTone('success')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['today-cards'] }),
        queryClient.invalidateQueries({ queryKey: ['memories'] }),
      ])
    },
    onError: () => {
      setSaveMessage('上传识别失败，请重试。')
      setSaveTone('error')
    },
  })

  const mentionMutation = useMutation({
    mutationFn: async (keyword: string) => {
      const response = await mention.query(keyword, 5)
      return response.data as MentionResult[]
    },
    onSuccess: (rows) => {
      setMentionResults(rows)
    },
    onError: () => {
      setMentionResults([])
    },
  })

  const reminderMutation = useMutation({
    mutationFn: async ({ title, remindAt }: { title: string; remindAt: string }) => {
      await reminders.create({ title, remind_at: remindAt, source: 'capture_bar' })
    },
    onSuccess: async () => {
      setSaveMessage('提醒已创建。')
      setSaveTone('success')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['today-cards'] }),
        queryClient.invalidateQueries({ queryKey: ['daily-brief-today'] }),
      ])
    },
    onError: () => {
      setSaveMessage('创建提醒失败，请稍后重试。')
      setSaveTone('error')
    },
  })

  const dismissCardMutation = useMutation({
    mutationFn: async (cardId: string) => {
      await today.dismiss(cardId)
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['today-cards'] })
    },
    onError: () => {
      setSaveMessage('暂时无法隐藏这条内容，请稍后重试。')
      setSaveTone('error')
    },
  })

  useEffect(() => {
    return () => {
      recognitionRef.current?.stop()
      recognitionRef.current = null
    }
  }, [])

  useEffect(() => {
    const keyword = taskTitle.trim()
    if (keyword.length < 2) {
      setMentionResults([])
      return
    }
    const timer = window.setTimeout(() => {
      mentionMutation.mutate(keyword)
    }, 240)
    return () => window.clearTimeout(timer)
  }, [taskTitle])

  const submitTask = () => {
    const title = taskTitle.trim()
    if (!title || intakeMutation.isPending) {
      return
    }
    setSaveMessage('')
    setSaveTone('info')
    setIntakeHint('')
    intakeMutation.mutate(title)
  }

  const handleSearch = () => {
    const title = taskTitle.trim()
    if (!title || searchMutation.isPending) {
      return
    }
    setSaveMessage('')
    setSaveTone('info')
    searchMutation.mutate(title)
  }

  const handleAsk = () => {
    const title = taskTitle.trim()
    if (!title) {
      return
    }
    onChat?.({ draft: title, autoSubmit: true })
  }

  const openUpload = () => {
    fileInputRef.current?.click()
  }

  const onUploadSelected = (file: File | null) => {
    if (!file) {
      return
    }
    uploadMutation.mutate({ files: [file], note: taskTitle.trim() || undefined })
  }

  const handleSetReminder = () => {
    const title = taskTitle.trim()
    if (!title || reminderMutation.isPending) {
      return
    }
    const remindAt = inferReminderAt(title)
    if (!remindAt) {
      setSaveMessage('未识别到明确时间，请补充“明天/后天/几点”。')
      setSaveTone('info')
      return
    }
    reminderMutation.mutate({ title, remindAt: remindAt.toISOString() })
  }

  const hasReminderHint = Boolean(inferReminderAt(taskTitle.trim()))

  const toggleVoiceInput = () => {
    const Constructor = (window as WindowWithSpeechRecognition).SpeechRecognition
      ?? (window as WindowWithSpeechRecognition).webkitSpeechRecognition
    if (!Constructor) {
      setSaveMessage('当前设备不支持语音输入，请改用键盘输入。')
      setSaveTone('error')
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
      setTaskTitle((current) => (current ? `${current} ${transcript}` : transcript))
      setSaveMessage('语音已转成文字。')
      setSaveTone('success')
    }
    recognition.onerror = (event) => {
      const reason = event?.error ? `（${event.error}）` : ''
      setSaveMessage(`语音输入失败，请检查麦克风权限${reason}`)
      setSaveTone('error')
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
      setSaveMessage('正在听你说话...')
      setSaveTone('info')
    } catch {
      setSaveMessage('无法启动语音输入，请检查浏览器与麦克风权限。')
      setSaveTone('error')
      setIsListening(false)
      recognitionRef.current = null
    }
  }

  return (
    <div className="px-5 pt-6 pb-4 space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        {/* 左上角 Logo - 点击进入记忆/洞察页 */}
        <button
          onClick={onMemory}
          className="ui-icon-btn bg-nesio-accentSoft"
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
          className="ui-icon-btn rounded-full bg-nesio-accentSoft overflow-hidden"
        >
          <div className="w-full h-full bg-nesio-accentLight flex items-center justify-center">
            <span className="text-sm font-bold text-nesio-accent">婧</span>
          </div>
        </button>
      </div>

      {/* Greeting */}
      <div>
        <h1 className="type-h1 text-nesio-ink">
          婧,晚上好。今天有 {cards.length} 条待处理内容，先看最重要的一条。
        </h1>
      </div>

      {/* Today Section Card */}
      <div className="ui-card-plain p-4 flex items-center gap-3 active:scale-[0.99] transition">
        <div className="w-10 h-10 rounded-xl bg-nesio-icon-bg flex items-center justify-center shrink-0">
          <svg className="w-5 h-5 text-nesio-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
            <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
            <polyline points="14 2 14 8 20 8"/>
          </svg>
        </div>
        <div className="flex-1 min-w-0">
          <div className="type-title text-nesio-ink">{primaryCard?.title ?? '今天这一段'}</div>
          <div className="type-caption text-nesio-muted truncate">
            {isLoading ? '正在同步今天卡片...' : briefQuery.data?.content ?? primaryCard?.body ?? `本地日历日 ${data?.local_day ?? '--'}`}
          </div>
        </div>
        <svg className="w-5 h-5 text-nesio-muted shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M9 18l6-6-6-6"/>
        </svg>
      </div>

      {/* Input */}
      <div className="flex items-center gap-3">
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*,.pdf,.txt,.md"
          className="hidden"
          onChange={(event) => {
            onUploadSelected(event.currentTarget.files?.[0] ?? null)
            event.currentTarget.value = ''
          }}
        />
        <button onClick={openUpload} className="ui-icon-btn rounded-full" aria-label="上传图片或文件">
          <IconPlus className="w-5 h-5" />
        </button>
        <div className="flex-1">
          <input
            type="text"
            value={taskTitle}
            onChange={(event) => setTaskTitle(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                submitTask()
              }
            }}
            placeholder="添加一个任务"
            className="ui-input-pill"
          />
        </div>
        <button onClick={toggleVoiceInput} className={`ui-icon-btn rounded-full ${isListening ? 'bg-nesio-accent text-white' : ''}`} aria-label="语音输入">
          <IconMic className="w-5 h-5" />
        </button>
        <button onClick={submitTask} className="ui-btn-primary rounded-full px-4" aria-label="保存">
          入库
        </button>
      </div>

      {taskTitle.trim() && (
        <div className="pl-14 flex flex-wrap gap-2">
          <button onClick={handleSearch} className="ui-chip">
            进入搜索
          </button>
          <button onClick={handleAsk} className="ui-chip ui-chip-active">
            直接问问
          </button>
          {hasReminderHint && (
            <button onClick={handleSetReminder} className="ui-chip">
              设提醒
            </button>
          )}
        </div>
      )}

      {mentionResults.length > 0 && (
        <div className="pl-14 flex flex-wrap gap-2">
          {mentionResults.map((row) => (
            <button
              key={row.id}
              onClick={() => setTaskTitle((current) => `${current.trim()} @${row.title}`.trim())}
              className="ui-chip"
            >
              @{row.title}
            </button>
          ))}
        </div>
      )}
      {saveMessage && (
        <p className={`-mt-3 pl-14 ${saveTone === 'error' ? 'ui-state-error' : saveTone === 'success' ? 'ui-state-success' : 'ui-state-info'}`}>
          {saveMessage}
        </p>
      )}
      {intakeHint && <p className="-mt-3 pl-14 ui-state-success">{intakeHint}</p>}

      {searchResults.length > 0 && (
        <div className="space-y-2">
          {searchResults.slice(0, 3).map((row) => (
            <div key={row.id} className="ui-card-plain px-3 py-2">
              <div className="type-body text-nesio-ink">{row.title}</div>
              <div className="type-caption text-nesio-muted mt-1">{row.type}</div>
            </div>
          ))}
        </div>
      )}

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
            <div className="type-body text-nesio-accent font-medium">现在</div>
            <div className="type-title text-nesio-ink mt-0.5">此刻,你感觉——</div>
            <div className="mt-2 flex gap-2 flex-wrap">
              {['平静', '焦虑', '疲惫', '充满能量'].map((m) => (
                <button
                  key={m}
                  onClick={() => setFeeling(m)}
                  className={`ui-chip transition ${
                    feeling === m
                      ? 'ui-chip-active'
                      : ''
                  }`}
                >
                  {m}
                </button>
              ))}
            </div>
          </div>
        </div>

        {cards.slice(1, 4).map((card) => {
          const safeBody = sanitizeBodyForDisplay(card.body)
          return (
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
                <div className="type-caption text-nesio-accent font-medium">{card.slot} · 严重度 {card.severity}</div>
                <div className="type-title text-nesio-ink mt-0.5">{card.title}</div>
                {safeBody && <div className="type-body text-nesio-muted mt-1">{safeBody}</div>}
              </div>
              <button
                type="button"
                onClick={() => dismissCardMutation.mutate(card.id)}
                disabled={dismissCardMutation.isPending}
                aria-label={`隐藏：${card.title}`}
                className="w-6 h-6 rounded-full bg-nesio-icon-bg flex items-center justify-center text-nesio-muted active:scale-90 transition disabled:opacity-50"
              >
                <IconX className="w-3 h-3" />
              </button>
            </div>
          </div>
          )
        })}
      </div>
    </div>
  )
}
