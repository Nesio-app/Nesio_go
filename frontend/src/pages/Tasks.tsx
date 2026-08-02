import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { tasks } from '../api/client'

export default function TasksPage() {
  const [status, setStatus] = useState('active')
  const [newTitle, setNewTitle] = useState('')
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['tasks', status],
    queryFn: () => tasks.list(status).then((r) => r.data),
  })

  const create = useMutation({
    mutationFn: (title: string) => tasks.create({ title, type: 'task' }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tasks'] })
      setNewTitle('')
    },
  })

  const update = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      tasks.update(id, { status }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks'] }),
  })

  if (isLoading) return <div className="text-center py-20">加载中...</div>

  return (
    <div>
      <h2 className="text-xl font-bold mb-4">任务</h2>
      <div className="flex gap-2 mb-4">
        {['active', 'later', 'archived', 'done'].map((s) => (
          <button
            key={s}
            onClick={() => setStatus(s)}
            className={`px-3 py-1 rounded-lg text-sm ${
              status === s
                ? 'bg-[var(--color-portal-blue)] text-white'
                : 'bg-white text-[var(--color-text-secondary)]'
            }`}
          >
            {s === 'active' ? '进行中' : s === 'later' ? '稍后' : s === 'archived' ? '归档' : '已完成'}
          </button>
        ))}
      </div>
      <div className="flex gap-2 mb-4">
        <input
          type="text"
          placeholder="新任务..."
          value={newTitle}
          onChange={(e) => setNewTitle(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && newTitle && create.mutate(newTitle)}
          className="flex-1 px-4 py-2 rounded-xl border border-[var(--color-border)] bg-white focus:outline-none focus:ring-2 focus:ring-[var(--color-portal-blue)]"
        />
        <button
          onClick={() => newTitle && create.mutate(newTitle)}
          className="btn-primary"
        >
          添加
        </button>
      </div>
      {data?.length === 0 && (
        <div className="text-center py-10 text-[var(--color-text-secondary)]">暂无任务</div>
      )}
      {data?.map((task: any) => (
        <div key={task.id} className="card flex items-center justify-between">
          <div>
            <p className="font-medium">{task.title}</p>
            {task.due_date && (
              <p className="text-xs text-[var(--color-text-secondary)]">
                截止: {new Date(task.due_date).toLocaleDateString('zh-CN')}
              </p>
            )}
          </div>
          <div className="flex gap-1">
            {task.status === 'active' && (
              <>
                <button
                  onClick={() => update.mutate({ id: task.id, status: 'done' })}
                  className="btn-ghost text-xs text-[var(--color-status-go)]"
                >
                  完成
                </button>
                <button
                  onClick={() => update.mutate({ id: task.id, status: 'later' })}
                  className="btn-ghost text-xs"
                >
                  稍后
                </button>
              </>
            )}
            {task.status !== 'active' && (
              <button
                onClick={() => update.mutate({ id: task.id, status: 'active' })}
                className="btn-ghost text-xs"
              >
                恢复
              </button>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}
