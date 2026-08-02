import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { today } from '../api/client'

export default function TodayPage() {
  const qc = useQueryClient()
  const { data, isLoading } = useQuery({
    queryKey: ['today'],
    queryFn: () => today.get().then((r) => r.data),
  })

  const dismiss = useMutation({
    mutationFn: (id: string) => today.dismiss(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['today'] }),
  })

  const done = useMutation({
    mutationFn: (id: string) => today.done(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['today'] }),
  })

  if (isLoading) return <div className="text-center py-20 text-[var(--color-text-secondary)]">加载中...</div>

  const cards = data?.cards || []

  return (
    <div>
      <h2 className="text-xl font-bold mb-1">{data?.local_day || '今日'}</h2>
      <p className="text-sm text-[var(--color-text-secondary)] mb-6">
        {cards.length} 张卡片
      </p>
      {cards.length === 0 && (
        <div className="text-center py-20 text-[var(--color-text-secondary)]">
          今天没有待办事项 🎉
        </div>
      )}
      {cards.map((card: any) => (
        <div
          key={card.id}
          className={`card card-${card.slot}`}
        >
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <h3 className="font-semibold text-[var(--color-text-primary)]">{card.title}</h3>
              {card.body && (
                <p className="text-sm text-[var(--color-text-secondary)] mt-1">{card.body}</p>
              )}
              <div className="flex items-center gap-2 mt-2">
                <span className={`text-xs px-2 py-0.5 rounded-full ${
                  card.severity === 3 ? 'bg-red-100 text-red-700' :
                  card.severity === 2 ? 'bg-yellow-100 text-yellow-700' :
                  'bg-green-100 text-green-700'
                }`}>
                  {card.severity === 3 ? '紧急' : card.severity === 2 ? '重要' : '普通'}
                </span>
                {card.action_label && (
                  <span className="text-xs text-[var(--color-portal-blue)]">{card.action_label}</span>
                )}
              </div>
            </div>
            <div className="flex gap-1 ml-3">
              <button
                onClick={() => done.mutate(card.id)}
                className="btn-ghost text-xs"
              >
                完成
              </button>
              <button
                onClick={() => dismiss.mutate(card.id)}
                className="btn-ghost text-xs"
              >
                跳过
              </button>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
