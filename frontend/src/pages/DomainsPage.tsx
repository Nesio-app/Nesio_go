import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  IconHeart, IconBox, IconCreditCard, IconCalendar,
  IconMapPin, IconActivity, IconUsers, IconHanger,
  IconUser, IconTarget, IconUtensils, IconSparkles,
  IconHome, IconBulb, IconPlay, IconSettings,
  IconMusic, IconGift, IconArrowLeft, IconPlus
} from '../icons'
import { domains as domainsApi, gmail } from '../api/client'

const domains = [
  { icon: IconHeart, label: '健康', color: 'text-nesio-accent', focus: '睡眠、恢复、体检', metric: '恢复指数 78', checklist: ['记录睡眠', '补水 2L', '安排体检提醒'] },
  { icon: IconBox, label: '物品', color: 'text-nesio-accent', focus: '库存、收纳、借出', metric: '待归位 6 件', checklist: ['整理桌面', '归档快递', '盘点常用物'] },
  { icon: IconCreditCard, label: '财务', color: 'text-nesio-accent', focus: '账单、预算、现金流', metric: '本周预算剩余 61%', checklist: ['核对账单', '复盘订阅', '记录支出'] },
  { icon: IconCalendar, label: '日程', color: 'text-nesio-accent', focus: '会议、提醒、出行', metric: '今天 4 个事件', checklist: ['确认明日行程', '预留缓冲', '同步家庭日历'] },
  { icon: IconMapPin, label: '足迹', color: 'text-nesio-accent', focus: '地点、路线、拜访', metric: '本周 3 个地点', checklist: ['整理常去地点', '补记路线', '复盘通勤时间'] },
  { icon: IconActivity, label: '健身', color: 'text-nesio-accent', focus: '训练、步数、心率', metric: '今日 6,820 步', checklist: ['晚间拉伸', '更新训练计划', '记录体感'] },
  { icon: IconUsers, label: '家务', color: 'text-nesio-accent', focus: '清洁、采购、家庭协作', metric: '待办 5 项', checklist: ['垃圾分类', '采购清单', '周末清洁排程'] },
  { icon: IconHanger, label: '衣橱', color: 'text-nesio-accent', focus: '穿搭、换季、清洗', metric: '本周搭配 4 套', checklist: ['挑选明日穿搭', '清洗深色衣物', '换季收纳'] },
  { icon: IconUser, label: '关系', color: 'text-nesio-accent', focus: '联络、纪念日、陪伴', metric: '本周 2 次主动联系', checklist: ['问候朋友', '安排家人通话', '补记生日'] },
  { icon: IconTarget, label: '成长', color: 'text-nesio-accent', focus: '阅读、学习、复盘', metric: '学习时长 3.2h', checklist: ['写学习笔记', '复盘本周', '设下个微目标'] },
  { icon: IconUtensils, label: '美味', color: 'text-nesio-accent', focus: '菜单、做饭、营养', metric: '已规划 2 餐', checklist: ['补充蛋白质', '准备早餐', '整理食材'] },
  { icon: IconSparkles, label: '镜子', color: 'text-nesio-accent', focus: '情绪、自评、复原力', metric: '情绪趋势 稳定', checklist: ['写一句感受', '标记触发点', '做 5 分钟呼吸'] },
  { icon: IconHome, label: '资产', color: 'text-nesio-accent', focus: '长期资产、清单、配置', metric: '月度复盘待完成', checklist: ['核对资产清单', '更新保险信息', '记录大额变动'] },
  { icon: IconBulb, label: '目标', color: 'text-nesio-accent', focus: '季度目标、推进节奏', metric: '季度完成 42%', checklist: ['拆成下周动作', '删掉低价值目标', '更新里程碑'] },
  { icon: IconPlay, label: '剧场', color: 'text-nesio-accent', focus: '观影、内容库、灵感', metric: '待看 7 条', checklist: ['收集片单', '写观后感', '整理高光片段'] },
  { icon: IconSettings, label: '运营', color: 'text-nesio-accent', focus: '系统维护、自动化、流程', metric: '自动化 3 条', checklist: ['复盘流程卡点', '清理低效提醒', '更新 SOP'] },
  { icon: IconMusic, label: '音乐', color: 'text-nesio-accent', focus: '歌单、练习、现场', metric: '本周新歌 12 首', checklist: ['整理歌单', '记录喜欢片段', '安排练习'] },
  { icon: IconGift, label: '奖励', color: 'text-nesio-accent', focus: '奖励机制、庆祝、恢复', metric: '本月已兑现 3 次', checklist: ['兑现一次奖励', '记录进步', '设计下个奖励'] },
] as const

const baseUrl = import.meta.env.BASE_URL || '/'
const domainCoverByLabel: Record<string, string> = {
  健康: `${baseUrl}ui/domains/IMG_0143.PNG`,
  物品: `${baseUrl}ui/domains/IMG_0144.PNG`,
  财务: `${baseUrl}ui/domains/IMG_0145.PNG`,
  日程: `${baseUrl}ui/domains/IMG_0146.PNG`,
  足迹: `${baseUrl}ui/domains/IMG_0147.PNG`,
  健身: `${baseUrl}ui/domains/IMG_0148.PNG`,
  家务: `${baseUrl}ui/domains/IMG_0149.PNG`,
  衣橱: `${baseUrl}ui/domains/IMG_0151.PNG`,
  关系: `${baseUrl}ui/domains/IMG_0152.PNG`,
  成长: `${baseUrl}ui/domains/IMG_0153.PNG`,
  美味: `${baseUrl}ui/domains/IMG_0154.PNG`,
  镜子: `${baseUrl}ui/domains/IMG_0155.PNG`,
  资产: `${baseUrl}ui/domains/IMG_0156.PNG`,
  目标: `${baseUrl}ui/domains/IMG_0157.PNG`,
  剧场: `${baseUrl}ui/domains/IMG_0158.PNG`,
  运营: `${baseUrl}ui/domains/IMG_0159.PNG`,
  音乐: `${baseUrl}ui/domains/IMG_0160.PNG`,
  奖励: `${baseUrl}ui/domains/IMG_0161.PNG`,
}

function sanitizeBodyForDisplay(body?: string | null): string {
  if (!body) {
    return ''
  }
  if (/AI not configured\. Add GEMINI_API_KEY/i.test(body)) {
    return ''
  }
  return body
}

interface Props {
  onToday?: () => void
  onMemory?: (domainLabel?: string) => void
  onChat?: () => void
  onSettings?: () => void
  onOpenItems?: () => void
}

export default function DomainsPage({ onToday, onMemory, onChat, onSettings, onOpenItems }: Props) {
  const queryClient = useQueryClient()
  const [selectedLabel, setSelectedLabel] = useState<(typeof domains)[number]['label'] | null>(null)
  const [scheduleSection, setScheduleSection] = useState<'calendar' | 'inbox' | 'sent'>('calendar')
  const [scheduleFilter, setScheduleFilter] = useState<'all' | 'todo' | 'noDue' | 'done' | 'notification' | 'important' | 'billing' | 'review'>('all')
  const [scheduleSearch, setScheduleSearch] = useState('')
  const [taskTitle, setTaskTitle] = useState('')
  const [memoryTitle, setMemoryTitle] = useState('')
  const [memoryBody, setMemoryBody] = useState('')
  const [gmailTo, setGmailTo] = useState('')
  const [gmailSubject, setGmailSubject] = useState('')
  const [gmailBody, setGmailBody] = useState('')
  const activeLabel = selectedLabel ?? '健康'
  const selectedDomain = domains.find((domain) => domain.label === activeLabel)
  const { data: overviewData } = useQuery({
    queryKey: ['domains-overview'],
    queryFn: async () => {
      const response = await domainsApi.overview()
      const rows = (response.data ?? []) as Array<{
        label: string
        task_count: number
        memory_count: number
        urgent_count: number
        latest_titles: string[] | null
      }>
      return rows.map((row) => ({
        ...row,
        latest_titles: Array.isArray(row.latest_titles) ? row.latest_titles : [],
      }))
    },
  })
  const detailQuery = useQuery({
    queryKey: ['domain-detail', activeLabel],
    enabled: Boolean(selectedLabel),
    queryFn: async () => {
      const response = await domainsApi.detail(activeLabel)
      return response.data as {
        domain: string
        tasks: Array<{ id: string; title: string; status: string; due_date?: string | null }>
        memory: Array<{ id: string; title: string; body?: string | null }>
        today: Array<{ id: string; title: string; severity: number; body?: string | null }>
      }
    },
  })
  const gmailInboxQuery = useQuery({
    queryKey: ['gmail-inbox', activeLabel, scheduleSection, scheduleSearch],
    enabled: selectedLabel === '日程' && scheduleSection !== 'calendar',
    queryFn: async () => {
      const response = await gmail.inbox({ box: scheduleSection === 'sent' ? 'sent' : 'inbox', q: scheduleSearch || undefined })
      return response.data as { messages: Array<{ id: string; from: string; subject: string; snippet: string }> }
    },
  })
  const selectedOverview = selectedLabel ? overviewData?.find((item) => item.label === selectedLabel) : undefined
  const overviewByLabel = useMemo(() => {
    const map = new Map<string, { task_count: number; memory_count: number }>()
    for (const item of overviewData ?? []) {
      map.set(item.label, { task_count: item.task_count, memory_count: item.memory_count })
    }
    return map
  }, [overviewData])
  const createTaskMutation = useMutation({
    mutationFn: async () => {
      if (!selectedLabel) {
        return
      }
      await domainsApi.createTask(selectedLabel, { title: taskTitle })
    },
    onSuccess: async () => {
      setTaskTitle('')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['domain-detail', activeLabel] }),
        queryClient.invalidateQueries({ queryKey: ['domains-overview'] }),
        queryClient.invalidateQueries({ queryKey: ['today-cards'] }),
      ])
    },
  })
  const createMemoryMutation = useMutation({
    mutationFn: async () => {
      if (!selectedLabel) {
        return
      }
      await domainsApi.createMemory(selectedLabel, { title: memoryTitle, body: memoryBody })
    },
    onSuccess: async () => {
      setMemoryTitle('')
      setMemoryBody('')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['domain-detail', activeLabel] }),
        queryClient.invalidateQueries({ queryKey: ['domains-overview'] }),
        queryClient.invalidateQueries({ queryKey: ['memories'] }),
      ])
    },
  })
  const deleteNodeMutation = useMutation({
    mutationFn: async ({ id }: { id: string }) => {
      if (!selectedLabel) {
        return
      }
      await domainsApi.deleteNode(selectedLabel, id)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['domain-detail', activeLabel] }),
        queryClient.invalidateQueries({ queryKey: ['domains-overview'] }),
      ])
    },
  })
  const gmailSendMutation = useMutation({
    mutationFn: async () => gmail.send({ to: gmailTo, subject: gmailSubject, body: gmailBody }),
    onSuccess: async () => {
      setGmailTo('')
      setGmailSubject('')
      setGmailBody('')
      await gmailInboxQuery.refetch()
    },
  })

  const scheduleCards = useMemo(() => {
    if (selectedLabel !== '日程') {
      return []
    }
    const tasks = detailQuery.data?.tasks ?? []
    const todayCards = detailQuery.data?.today ?? []
    const mapped = [
      ...todayCards.map((item) => ({
        id: item.id,
        title: item.title,
        subtitle: item.body ?? '日程提醒',
        meta: item.severity >= 3 ? '提醒' : '日程',
        date: '今天',
        section: item.severity >= 3 ? 'important' : 'notification',
      })),
      ...tasks.map((task) => ({
        id: task.id,
        title: task.title,
        subtitle: task.status === 'done' ? '已完成' : '待办事项',
        meta: task.due_date ? '待办' : '无截止日期',
        date: task.due_date ? new Date(task.due_date).toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric' }) : '稍后',
        section: task.status === 'done' ? 'done' : task.due_date ? 'todo' : 'noDue',
      })),
    ]
    return mapped.filter((item) => {
      const searchHit = !scheduleSearch.trim() || `${item.title} ${item.subtitle}`.toLowerCase().includes(scheduleSearch.trim().toLowerCase())
      const filterHit = scheduleFilter === 'all' || item.section === scheduleFilter
      return searchHit && filterHit
    })
  }, [detailQuery.data, scheduleFilter, scheduleSearch, selectedLabel])

  const scheduleMailCards = useMemo(() => {
    const messages = gmailInboxQuery.data?.messages ?? []
    return messages.filter((message) => {
      const searchHit = !scheduleSearch.trim() || `${message.subject} ${message.from} ${message.snippet}`.toLowerCase().includes(scheduleSearch.trim().toLowerCase())
      if (!searchHit) {
        return false
      }
      if (scheduleFilter === 'all') {
        return true
      }
      if (scheduleFilter === 'important') {
        return /important|urgent|action|required/i.test(`${message.subject} ${message.snippet}`)
      }
      if (scheduleFilter === 'notification') {
        return /notification|notice|reminder/i.test(`${message.subject} ${message.snippet}`)
      }
      if (scheduleFilter === 'billing') {
        return /bill|invoice|receipt|refund|order|payment/i.test(`${message.subject} ${message.snippet}`)
      }
      if (scheduleFilter === 'review') {
        return /review|comment|feedback/i.test(`${message.subject} ${message.snippet}`)
      }
      return true
    })
  }, [gmailInboxQuery.data, scheduleFilter, scheduleSearch])

  const scheduleTabCounts = {
    calendar: (detailQuery.data?.tasks?.length ?? 0) + (detailQuery.data?.today?.length ?? 0),
    inbox: gmailInboxQuery.data?.messages?.length ?? selectedOverview?.memory_count ?? 0,
    sent: scheduleSection === 'sent' ? gmailInboxQuery.data?.messages?.length ?? 0 : 0,
  }

  const renderScheduleWorkspace = () => (
    <div className="space-y-5">
      <div className="flex items-center justify-between px-1">
        <button onClick={() => setSelectedLabel(null)} className="ui-icon-btn w-10 h-10 rounded-full bg-nesio-accentSoft text-nesio-accent">
          <IconArrowLeft className="w-8 h-8" />
        </button>
        <div className="type-h1 text-nesio-ink">日程</div>
        <button onClick={() => onToday?.()} className="ui-btn-secondary rounded-full px-4">
          今天
        </button>
      </div>

      <div className="rounded-3xl bg-nesio-accentSoft p-2 flex items-center gap-2">
        {[
          { key: 'calendar', label: '日历项', count: scheduleTabCounts.calendar },
          { key: 'inbox', label: '收件', count: scheduleTabCounts.inbox },
          { key: 'sent', label: '发件', count: scheduleTabCounts.sent },
        ].map((item) => (
          <button
            key={item.key}
            onClick={() => setScheduleSection(item.key as 'calendar' | 'inbox' | 'sent')}
            className={`flex-1 rounded-2xl px-4 py-3 type-title transition ${scheduleSection === item.key ? 'bg-white text-nesio-accent shadow-card' : 'text-nesio-muted'}`}
          >
            {item.label} {item.count}
          </button>
        ))}
      </div>

      <input
        value={scheduleSearch}
        onChange={(e) => setScheduleSearch(e.target.value)}
        placeholder={scheduleSection === 'calendar' ? '搜日程:标题、地点、日历名..' : '搜邮件:标题、发件人、正文..'}
        className="ui-input"
      />

      <div className="flex gap-4 overflow-x-auto pb-1 scrollbar-hide">
        {(scheduleSection === 'calendar'
          ? [
              { key: 'all', label: '全部', count: scheduleTabCounts.calendar },
              { key: 'todo', label: '待办', count: (detailQuery.data?.tasks ?? []).filter((item) => item.status !== 'done' && item.due_date).length },
              { key: 'noDue', label: '无截止日期', count: (detailQuery.data?.tasks ?? []).filter((item) => !item.due_date).length },
              { key: 'done', label: '看已完成的提醒', count: (detailQuery.data?.tasks ?? []).filter((item) => item.status === 'done').length },
            ]
          : [
              { key: 'all', label: '全部', count: scheduleMailCards.length },
              { key: 'notification', label: '通知', count: scheduleMailCards.filter((item) => /notification|notice|reminder/i.test(`${item.subject} ${item.snippet}`)).length },
              { key: 'important', label: '重要', count: scheduleMailCards.filter((item) => /important|urgent|action|required/i.test(`${item.subject} ${item.snippet}`)).length },
              ...(scheduleSection === 'sent'
                ? [{ key: 'review', label: '评测', count: scheduleMailCards.filter((item) => /review|comment|feedback/i.test(`${item.subject} ${item.snippet}`)).length }]
                : [{ key: 'billing', label: '订单', count: scheduleMailCards.filter((item) => /bill|invoice|receipt|refund|order|payment/i.test(`${item.subject} ${item.snippet}`)).length }]),
            ]).map((chip) => (
          <button
            key={chip.key}
            onClick={() => setScheduleFilter(chip.key as typeof scheduleFilter)}
            className={`whitespace-nowrap ui-chip transition ${scheduleFilter === chip.key ? 'ui-chip-active' : ''}`}
          >
            {chip.label} {chip.count}
          </button>
        ))}
        {scheduleSection === 'sent' && (
          <button className="ui-chip border-dashed text-nesio-accent">
            <IconPlus className="w-7 h-7 inline-block" />
          </button>
        )}
      </div>

      {scheduleSection === 'sent' && (
        <button className="ui-btn-primary w-full" onClick={() => gmailTo || gmailSubject || gmailBody ? gmailSendMutation.mutate() : null}>
          写一封
        </button>
      )}

      {scheduleSection === 'sent' && (
        <div className="ui-card-plain p-5 space-y-3">
          <input value={gmailTo} onChange={(e) => setGmailTo(e.target.value)} placeholder="发给谁" className="ui-input bg-nesio-bg" />
          <input value={gmailSubject} onChange={(e) => setGmailSubject(e.target.value)} placeholder="主题" className="ui-input bg-nesio-bg" />
          <textarea value={gmailBody} onChange={(e) => setGmailBody(e.target.value)} placeholder="正文" className="ui-input min-h-28 bg-nesio-bg" />
        </div>
      )}

      <div className="space-y-4">
        {scheduleSection === 'calendar' && scheduleCards.map((item) => (
          <div key={item.id} className="ui-card-plain p-5">
            <div className="flex items-start justify-between gap-4">
              <div className="min-w-0 flex-1">
                <div className="type-title text-nesio-ink truncate">{item.title}</div>
                <div className="mt-3 flex items-center gap-2 type-caption text-nesio-muted">
                  <span>{item.meta}</span>
                  {item.section === 'important' && <span className="rounded-full bg-nesio-accentSoft px-3 py-1 text-nesio-accent">提醒</span>}
                </div>
                <div className="mt-2 type-caption text-nesio-muted truncate">{item.subtitle}</div>
              </div>
              <div className="text-right type-caption text-nesio-muted whitespace-pre-line">{item.date}</div>
            </div>
          </div>
        ))}

        {scheduleSection !== 'calendar' && scheduleMailCards.map((message) => (
          <div key={message.id} className="ui-card-plain p-5">
            <div className="flex items-start justify-between gap-4">
              <div className="min-w-0 flex-1">
                <div className="type-title text-nesio-ink truncate">{message.subject || '(无主题)'}</div>
                <div className="mt-2 type-caption text-nesio-muted truncate">{scheduleSection === 'sent' ? `发给 ${message.from}` : message.from}</div>
                <div className="mt-2 type-caption text-nesio-muted truncate">{message.snippet}</div>
              </div>
              <div className="text-right type-caption text-nesio-muted">8月2日</div>
            </div>
          </div>
        ))}

        {scheduleSection !== 'calendar' && gmailInboxQuery.isError && (
          <div className="ui-card-plain p-5 type-body text-nesio-accent">
            还没有可用的 Gmail connector。先为 `gmail` 写入可用的 `access_token`。
          </div>
        )}
      </div>
    </div>
  )

  if (selectedLabel === '日程') {
    return renderScheduleWorkspace()
  }

  if (!selectedLabel) {
    return (
      <div className="px-5 pt-6 pb-6 space-y-5">
        <div>
          <div className="text-2xl font-bold text-nesio-ink">18 个核心领域</div>
          <div className="text-sm text-nesio-muted mt-1">每个入口现在都有一个可落地的领域看板，而不只是空壳。</div>
        </div>
        <div className="grid grid-cols-4 gap-3">
          {domains.map((d) => {
            const overview = overviewByLabel.get(d.label)
            return (
              <button
                key={d.label}
                onClick={() => {
                  if (d.label === '物品') {
                    onOpenItems?.()
                    return
                  }
                  setSelectedLabel(d.label)
                }}
                className="flex flex-col items-center gap-2 py-3 rounded-2xl active:scale-95 transition"
              >
                <div className="relative w-16 h-16 rounded-2xl overflow-hidden shadow-card bg-white">
                  <img src={domainCoverByLabel[d.label]} alt={d.label} className="w-full h-full object-cover" />
                  <div className="absolute inset-0 bg-black/25 flex items-center justify-center">
                    <d.icon className="w-7 h-7 text-white" />
                  </div>
                </div>
                <span className="text-sm text-nesio-ink font-medium">{d.label}</span>
                <span className="type-caption text-nesio-muted">
                  {overview ? `任务${overview.task_count} 记忆${overview.memory_count}` : d.metric}
                </span>
              </button>
            )
          })}
        </div>
      </div>
    )
  }

  return (
    <div className="px-5 pt-6 pb-6 space-y-5">
      <div className="flex items-center justify-between px-1">
        <button onClick={() => setSelectedLabel(null)} className="ui-icon-btn w-10 h-10 rounded-full bg-nesio-accentSoft text-nesio-accent">
          <IconArrowLeft className="w-8 h-8" />
        </button>
        <div className="type-h1 text-nesio-ink">{selectedDomain?.label ?? '领域看板'}</div>
        <button onClick={() => onToday?.()} className="ui-btn-secondary rounded-full px-4">
          今天
        </button>
      </div>

      <div className="nesio-card p-5 space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <div className="text-xl font-bold text-nesio-ink">{selectedDomain?.label ?? ''}</div>
            <div className="text-sm text-nesio-muted mt-1">{selectedDomain?.focus ?? ''}</div>
          </div>
          <div className="px-3 py-2 rounded-full bg-nesio-accentSoft text-sm text-nesio-accent">
            {selectedOverview ? `任务 ${selectedOverview.task_count} · 记忆 ${selectedOverview.memory_count}` : selectedDomain?.metric}
          </div>
        </div>

        <div className="grid grid-cols-1 gap-3">
          {(selectedDomain?.checklist ?? []).map((item) => (
            <div key={item} className="rounded-2xl border border-nesio-border px-4 py-3 bg-white/70">
              <div className="text-sm font-medium text-nesio-ink">{item}</div>
            </div>
          ))}
        </div>

        <div className="rounded-2xl bg-nesio-icon-bg px-4 py-4">
          <div className="text-sm text-nesio-muted">领域建议</div>
          <div className="text-base text-nesio-ink mt-1">
            先从「{selectedDomain?.checklist?.[0] ?? '第一步'}」开始，把这个领域的第一步变成今天卡片，再逐步沉淀到记忆和任务里。
          </div>
          {selectedOverview && Array.isArray(selectedOverview.latest_titles) && selectedOverview.latest_titles.length > 0 && (
            <div className="mt-3 space-y-1">
              {selectedOverview.latest_titles.map((title) => (
                <div key={title} className="text-sm text-nesio-muted">• {title}</div>
              ))}
            </div>
          )}
        </div>

        <div className="flex flex-wrap gap-2">
          <button onClick={() => onChat?.()} className="ui-btn-secondary">
            问问这个板块
          </button>
          <button onClick={() => onMemory?.(selectedLabel)} className="ui-btn-secondary">
            查看相关记忆
          </button>
          {selectedLabel === '运营' && (
            <button onClick={() => onSettings?.()} className="ui-btn-secondary">
              打开系统设置
            </button>
          )}
        </div>

        {selectedLabel === '物品' && (
          <button onClick={() => onOpenItems?.()} className="ui-btn-primary w-full">
            进入物品聚合页
          </button>
        )}

        <div className="grid grid-cols-1 gap-4">
          <div className="rounded-2xl border border-nesio-border p-4 bg-white/70 space-y-3">
            <div className="text-sm font-semibold text-nesio-ink">新增任务</div>
            <input value={taskTitle} onChange={(e) => setTaskTitle(e.target.value)} placeholder={`给${selectedLabel}新增一个任务`} className="w-full rounded-xl border border-nesio-border bg-white px-3 py-2 text-sm outline-none" />
            <button onClick={() => taskTitle.trim() && createTaskMutation.mutate()} className="px-4 py-2 rounded-full bg-nesio-accent text-white text-sm">创建任务</button>
          </div>

          <div className="rounded-2xl border border-nesio-border p-4 bg-white/70 space-y-3">
            <div className="text-sm font-semibold text-nesio-ink">新增记忆</div>
            <input value={memoryTitle} onChange={(e) => setMemoryTitle(e.target.value)} placeholder={`给${selectedLabel}新增一条记忆标题`} className="w-full rounded-xl border border-nesio-border bg-white px-3 py-2 text-sm outline-none" />
            <textarea value={memoryBody} onChange={(e) => setMemoryBody(e.target.value)} placeholder="补充正文" className="w-full min-h-24 rounded-xl border border-nesio-border bg-white px-3 py-2 text-sm outline-none" />
            <button onClick={() => memoryTitle.trim() && createMemoryMutation.mutate()} className="px-4 py-2 rounded-full bg-nesio-accent text-white text-sm">创建记忆</button>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className="rounded-2xl border border-nesio-border p-4 bg-white/70 space-y-3">
            <div className="text-sm font-semibold text-nesio-ink">领域任务</div>
            {(detailQuery.data?.tasks ?? []).slice(0, 6).map((task) => (
              <div key={task.id} className="flex items-center justify-between gap-3 rounded-xl bg-white px-3 py-2">
                <div>
                  <div className="text-sm text-nesio-ink">{task.title}</div>
                  <div className="text-xs text-nesio-muted">{task.status}</div>
                </div>
                <button onClick={() => deleteNodeMutation.mutate({ id: task.id })} className="text-xs text-nesio-muted">删除</button>
              </div>
            ))}
          </div>

          <div className="rounded-2xl border border-nesio-border p-4 bg-white/70 space-y-3">
            <div className="text-sm font-semibold text-nesio-ink">领域记忆</div>
            {(detailQuery.data?.memory ?? []).slice(0, 6).map((item) => (
              <div key={item.id} className="rounded-xl bg-white px-3 py-2">
                <div className="text-sm text-nesio-ink">{item.title}</div>
                {sanitizeBodyForDisplay(item.body) && <div className="text-xs text-nesio-muted mt-1">{sanitizeBodyForDisplay(item.body)}</div>}
              </div>
            ))}
          </div>
        </div>

      </div>
    </div>
  )
}
