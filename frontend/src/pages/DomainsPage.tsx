import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  IconHeart, IconBox, IconCreditCard, IconCalendar,
  IconMapPin, IconActivity, IconUsers, IconHanger,
  IconUser, IconTarget, IconUtensils, IconSparkles,
  IconHome, IconBulb, IconPlay, IconSettings,
  IconMusic, IconGift
} from '../icons'
import { domains as domainsApi } from '../api/client'

const domains = [
  { icon: IconHeart, label: '健康', color: 'text-red-400', focus: '睡眠、恢复、体检', metric: '恢复指数 78', checklist: ['记录睡眠', '补水 2L', '安排体检提醒'] },
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

interface Props {
  onToday?: () => void
  onMemory?: () => void
  onChat?: () => void
}

export default function DomainsPage({ onToday, onMemory, onChat }: Props) {
  // avoid unused parameter errors when embedded without handlers
  void onToday
  void onMemory
  void onChat
  const [selectedLabel, setSelectedLabel] = useState<(typeof domains)[number]['label']>('健康')
  const selectedDomain = domains.find((domain) => domain.label === selectedLabel) ?? domains[0]
  const { data: overviewData } = useQuery({
    queryKey: ['domains-overview'],
    queryFn: async () => {
      const response = await domainsApi.overview()
      return response.data as Array<{ label: string; task_count: number; memory_count: number; urgent_count: number; latest_titles: string[] }>
    },
  })
  const selectedOverview = overviewData?.find((item) => item.label === selectedLabel)

  return (
    <div className="px-5 pt-6 pb-6 space-y-5">
      <div>
        <div className="text-2xl font-bold text-nesio-ink">18 个核心领域</div>
        <div className="text-sm text-nesio-muted mt-1">每个入口现在都有一个可落地的领域看板，而不只是空壳。</div>
      </div>
      <div className="grid grid-cols-4 gap-3">
        {domains.map((d) => (
          <button
            key={d.label}
            onClick={() => setSelectedLabel(d.label)}
            className={`flex flex-col items-center gap-2 py-3 rounded-2xl active:scale-95 transition ${selectedLabel === d.label ? 'bg-white shadow-card' : ''}`}
          >
            <div className="w-16 h-16 rounded-2xl bg-white shadow-card flex items-center justify-center">
              <d.icon className={`w-7 h-7 ${d.color}`} />
            </div>
            <span className="text-sm text-nesio-ink font-medium">{d.label}</span>
          </button>
        ))}
      </div>

      <div className="nesio-card p-5 space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <div className="text-xl font-bold text-nesio-ink">{selectedDomain.label}</div>
            <div className="text-sm text-nesio-muted mt-1">{selectedDomain.focus}</div>
          </div>
          <div className="px-3 py-2 rounded-full bg-nesio-accentSoft text-sm text-nesio-accent">
              {selectedOverview ? `任务 ${selectedOverview.task_count} · 记忆 ${selectedOverview.memory_count}` : selectedDomain.metric}
          </div>
        </div>

        <div className="grid grid-cols-1 gap-3">
          {selectedDomain.checklist.map((item) => (
            <div key={item} className="rounded-2xl border border-nesio-border px-4 py-3 bg-white/70">
              <div className="text-sm font-medium text-nesio-ink">{item}</div>
            </div>
          ))}
        </div>

        <div className="rounded-2xl bg-nesio-icon-bg px-4 py-4">
          <div className="text-sm text-nesio-muted">领域建议</div>
          <div className="text-base text-nesio-ink mt-1">
            先从「{selectedDomain.checklist[0]}」开始，把这个领域的第一步变成今天卡片，再逐步沉淀到记忆和任务里。
          </div>
          {selectedOverview && selectedOverview.latest_titles.length > 0 && (
            <div className="mt-3 space-y-1">
              {selectedOverview.latest_titles.map((title) => (
                <div key={title} className="text-sm text-nesio-muted">• {title}</div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
