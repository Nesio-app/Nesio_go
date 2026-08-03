interface DuplicateItem {
  id?: string
  name?: string
  room_name?: string | null
  container_name?: string | null
  match_type?: string
}

interface Props {
  duplicates: DuplicateItem[]
  onUpdate: (targetItemId: string) => void
  onCreateNew: () => void
  onCancel: () => void
  isPending?: boolean
}

function matchTypeLabel(matchType?: string): string {
  if (matchType === 'visual') {
    return '看起来一样'
  }
  if (matchType === 'semantic') {
    return '看起来相似'
  }
  return '名字相似'
}

export default function DuplicateAlert({ duplicates, onUpdate, onCreateNew, onCancel, isPending = false }: Props) {
  if (duplicates.length === 0) {
    return null
  }

  return (
    <div className="rounded-2xl border border-nesio-accentLight bg-nesio-accentSoft p-4 space-y-3">
      <div className="type-title text-nesio-accent">这个物品家里已经有了</div>

      <div className="space-y-2">
        {duplicates.slice(0, 3).map((dup, index) => (
          <div key={`${dup.id ?? index}`} className="rounded-xl bg-white px-3 py-2">
            <div className="type-body text-nesio-ink">{dup.name ?? '已存在物品'}</div>
            <div className="type-caption text-nesio-muted mt-1">
              放在：{dup.room_name ?? '未设置房间'} · {dup.container_name ?? '未设置容器'}
            </div>
            <div className="type-caption text-nesio-muted mt-1">匹配方式：{matchTypeLabel(dup.match_type)}</div>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 gap-2">
        <button
          onClick={() => {
            const targetId = duplicates.find((item) => item.id)?.id
            if (targetId) {
              onUpdate(targetId)
            }
          }}
          disabled={isPending || !duplicates.find((item) => item.id)?.id}
          className="ui-btn-primary w-full"
        >
          {isPending ? '处理中...' : '更新现有记录（数量+1）'}
        </button>
        <button onClick={onCreateNew} disabled={isPending} className="ui-btn-secondary w-full">
          这是新的（创建新记录）
        </button>
        <button onClick={onCancel} disabled={isPending} className="ui-btn-ghost w-full">
          取消
        </button>
      </div>
    </div>
  )
}
