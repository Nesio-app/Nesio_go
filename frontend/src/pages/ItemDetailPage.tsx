import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { containers, items, rooms } from '../api/client'

interface Props {
  itemId: string
  onBack: () => void
}

interface ItemDetail {
  id: string
  name: string
  body?: string
  room_id?: string | null
  container_id?: string | null
  room_name?: string | null
  room_icon?: string | null
  container_name?: string | null
  container_icon?: string | null
  expiry_date?: string | null
  days_until_expiry?: number | null
  is_document?: boolean
  quantity?: number
  unit?: string | null
  primary_image_url?: string | null
}

function dateInputValue(iso?: string | null): string {
  if (!iso) {
    return ''
  }
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return ''
  }
  return date.toISOString().slice(0, 10)
}

export default function ItemDetailPage({ itemId, onBack }: Props) {
  const queryClient = useQueryClient()
  const [isEditing, setIsEditing] = useState(false)
  const [nameDraft, setNameDraft] = useState('')
  const [bodyDraft, setBodyDraft] = useState('')
  const [roomIdDraft, setRoomIdDraft] = useState('')
  const [containerIdDraft, setContainerIdDraft] = useState('')
  const [expiryDateDraft, setExpiryDateDraft] = useState('')
  const [quantityDraft, setQuantityDraft] = useState(1)

  const itemQuery = useQuery({
    queryKey: ['item-detail', itemId],
    queryFn: async () => (await items.get(itemId)).data as ItemDetail,
  })

  const roomsQuery = useQuery({
    queryKey: ['rooms'],
    queryFn: async () => (await rooms.list()).data as Array<{ id: string; name: string; icon: string }>,
  })

  const containersQuery = useQuery({
    queryKey: ['containers', roomIdDraft || itemQuery.data?.room_id || 'all'],
    queryFn: async () => {
      const roomId = roomIdDraft || itemQuery.data?.room_id || undefined
      return (await containers.list(roomId || undefined)).data as Array<{ id: string; name: string; icon: string }>
    },
  })

  const updateMutation = useMutation({
    mutationFn: async () => {
      await items.update(itemId, {
        name: nameDraft,
        body: bodyDraft || undefined,
        room_id: roomIdDraft || undefined,
        container_id: containerIdDraft || undefined,
        expiry_date: expiryDateDraft || undefined,
        quantity: quantityDraft,
      })
    },
    onSuccess: async () => {
      setIsEditing(false)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['item-detail', itemId] }),
        queryClient.invalidateQueries({ queryKey: ['items'] }),
        queryClient.invalidateQueries({ queryKey: ['items-expiring'] }),
        queryClient.invalidateQueries({ queryKey: ['items-documents'] }),
      ])
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async () => {
      await items.remove(itemId)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['items'] }),
        queryClient.invalidateQueries({ queryKey: ['items-expiring'] }),
        queryClient.invalidateQueries({ queryKey: ['items-documents'] }),
      ])
      onBack()
    },
  })

  const snoozeMutation = useMutation({
    mutationFn: async () => {
      await items.snoozeExpiry(itemId)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['item-detail', itemId] }),
        queryClient.invalidateQueries({ queryKey: ['items-expiring'] }),
      ])
    },
  })

  const item = itemQuery.data

  const locationText = useMemo(() => {
    if (!item) {
      return ''
    }
    const room = item.room_name ?? '未设置房间'
    const container = item.container_name ?? '未设置容器'
    return `${room} -> ${container}`
  }, [item])

  if (itemQuery.isLoading) {
    return <div className="p-6 type-body text-nesio-muted">加载物品详情中...</div>
  }

  if (!item) {
    return (
      <div className="p-6 space-y-3">
        <div className="type-body text-nesio-accent">物品不存在或已删除。</div>
        <button onClick={onBack} className="ui-btn-secondary">返回</button>
      </div>
    )
  }

  return (
    <div className="px-5 pt-6 pb-6 space-y-4">
      <div className="flex items-center justify-between">
        <button onClick={onBack} className="ui-btn-ghost">返回</button>
        <div className="type-title text-nesio-ink">物品详情</div>
        <button
          onClick={() => {
            if (!isEditing) {
              setNameDraft(item.name)
              setBodyDraft(item.body ?? '')
              setRoomIdDraft(item.room_id ?? '')
              setContainerIdDraft(item.container_id ?? '')
              setExpiryDateDraft(dateInputValue(item.expiry_date))
              setQuantityDraft(item.quantity ?? 1)
            }
            setIsEditing((v) => !v)
          }}
          className="ui-btn-secondary"
        >
          {isEditing ? '取消编辑' : '编辑'}
        </button>
      </div>

      {item.primary_image_url && (
        <img src={item.primary_image_url} alt={item.name} className="w-full h-52 object-cover rounded-2xl" />
      )}

      <div className="ui-card p-4 space-y-3">
        {isEditing ? (
          <>
            <input value={nameDraft} onChange={(e) => setNameDraft(e.target.value)} className="ui-input" placeholder="物品名" />
            <input value={bodyDraft} onChange={(e) => setBodyDraft(e.target.value)} className="ui-input" placeholder="描述" />
            <select value={roomIdDraft} onChange={(e) => { setRoomIdDraft(e.target.value); setContainerIdDraft('') }} className="ui-input">
              <option value="">未设置房间</option>
              {roomsQuery.data?.map((room) => (
                <option key={room.id} value={room.id}>{room.name}</option>
              ))}
            </select>
            <select value={containerIdDraft} onChange={(e) => setContainerIdDraft(e.target.value)} className="ui-input">
              <option value="">未设置容器</option>
              {containersQuery.data?.map((container) => (
                <option key={container.id} value={container.id}>{container.name}</option>
              ))}
            </select>
            <input type="date" value={expiryDateDraft} onChange={(e) => setExpiryDateDraft(e.target.value)} className="ui-input" />
            <input
              type="number"
              min={1}
              value={quantityDraft}
              onChange={(e) => setQuantityDraft(Math.max(1, Number(e.target.value || '1')))}
              className="ui-input"
            />
            <button onClick={() => updateMutation.mutate()} disabled={updateMutation.isPending} className="ui-btn-primary w-full">
              {updateMutation.isPending ? '保存中...' : '保存修改'}
            </button>
          </>
        ) : (
          <>
            <div className="type-h2 text-nesio-ink">{item.name}</div>
            {item.body && <div className="type-body text-nesio-muted">{item.body}</div>}
            <div className="type-body text-nesio-muted">位置：{locationText}</div>
            <div className="type-body text-nesio-muted">数量：{item.quantity ?? 1} {item.unit ?? '个'}</div>
            {item.expiry_date && (
              <div className="type-body text-nesio-muted">
                有效期：{dateInputValue(item.expiry_date)}
                {typeof item.days_until_expiry === 'number' && (
                  <span>（{item.days_until_expiry >= 0 ? `${item.days_until_expiry} 天后到期` : `已过期 ${-item.days_until_expiry} 天`}）</span>
                )}
              </div>
            )}
            {item.is_document && <div className="type-caption text-nesio-accent">证件类型物品</div>}
          </>
        )}
      </div>

      <div className="grid grid-cols-1 gap-2">
        <button onClick={() => snoozeMutation.mutate()} disabled={snoozeMutation.isPending} className="ui-btn-secondary w-full">
          {snoozeMutation.isPending ? '处理中...' : '推迟到期提醒（+7天）'}
        </button>
        <button onClick={() => deleteMutation.mutate()} disabled={deleteMutation.isPending} className="ui-btn-ghost w-full">
          {deleteMutation.isPending ? '删除中...' : '删除物品'}
        </button>
      </div>
    </div>
  )
}
