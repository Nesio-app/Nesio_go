import { useRef, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { items } from '../api/client'

interface WhereIsResult {
  type?: 'found' | 'multiple' | 'not_found'
  answer?: string
  item?: { id: string; name: string; room_name?: string | null; container_name?: string | null; primary_image_url?: string | null }
  items?: Array<{ id: string; name: string; room_name?: string | null; container_name?: string | null; primary_image_url?: string | null }>
}

interface Props {
  onOpenItem?: (itemId: string) => void
}

export default function WhereIs({ onOpenItem }: Props) {
  const [query, setQuery] = useState('')
  const [result, setResult] = useState<WhereIsResult | null>(null)
  const photoInputRef = useRef<HTMLInputElement>(null)

  const whereMutation = useMutation({
    mutationFn: async (q: string) => (await items.whereIs(q)).data as WhereIsResult,
    onSuccess: (data) => setResult(data),
  })

  const wherePhotoMutation = useMutation({
    mutationFn: async (file: File) => (await items.whereIsPhoto(file)).data as WhereIsResult,
    onSuccess: (data) => setResult(data),
  })

  return (
    <div className="ui-card p-4 space-y-3">
      <div className="type-title text-nesio-ink">拍照告诉我放在哪</div>
      <div className="flex gap-2">
        <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="我的护照在哪？" className="ui-input flex-1" />
        <button
          onClick={() => whereMutation.mutate(query)}
          disabled={whereMutation.isPending || !query.trim()}
          className="ui-btn-primary"
        >
          查询
        </button>
      </div>

      <input
        ref={photoInputRef}
        type="file"
        accept="image/*"
        capture="environment"
        className="hidden"
        onChange={(event) => {
          const file = event.currentTarget.files?.[0]
          if (file) {
            wherePhotoMutation.mutate(file)
          }
          event.currentTarget.value = ''
        }}
      />
      <button onClick={() => photoInputRef.current?.click()} className="ui-btn-ghost" disabled={wherePhotoMutation.isPending}>
        {wherePhotoMutation.isPending ? '识别中...' : '拍照查询'}
      </button>

      {result && (
        <div className="space-y-2">
          <div className="type-body text-nesio-muted">{result.answer ?? '未找到位置'}</div>
          {result.type === 'found' && result.item && (
            <button onClick={() => onOpenItem?.(result.item!.id)} className="ui-btn-secondary w-full">查看详情</button>
          )}
          {result.type === 'multiple' && (result.items ?? []).length > 0 && (
            <div className="space-y-2">
              {(result.items ?? []).map((entry) => (
                <button key={entry.id} onClick={() => onOpenItem?.(entry.id)} className="ui-btn-ghost w-full justify-start">
                  {entry.name} · {entry.room_name ?? '未设置房间'}
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
