import { useMemo, useRef, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { containers, items, rooms } from '../api/client'

export default function ItemsPage() {
  const [selectedRoom, setSelectedRoom] = useState<string>('')
  const [selectedContainer, setSelectedContainer] = useState<string>('')
  const [query, setQuery] = useState('')
  const [whereQuery, setWhereQuery] = useState('')
  const [whereAnswer, setWhereAnswer] = useState('')
  const wherePhotoInputRef = useRef<HTMLInputElement>(null)

  const roomsQuery = useQuery({
    queryKey: ['rooms'],
    queryFn: async () => (await rooms.list()).data as Array<{ id: string; name: string; icon: string }>,
  })
  const containersQuery = useQuery({
    queryKey: ['containers', selectedRoom],
    queryFn: async () => (await containers.list(selectedRoom || undefined)).data as Array<{ id: string; name: string; icon: string }>,
  })
  const itemsQuery = useQuery({
    queryKey: ['items', selectedRoom, selectedContainer, query],
    queryFn: async () => (await items.list({ room: selectedRoom || undefined, container: selectedContainer || undefined, q: query || undefined })).data as Array<any>,
  })
  const expiringQuery = useQuery({
    queryKey: ['items-expiring'],
    queryFn: async () => (await items.expiring()).data as Array<any>,
  })
  const docsQuery = useQuery({
    queryKey: ['items-documents'],
    queryFn: async () => (await items.documents()).data as Array<any>,
  })

  const whereMutation = useMutation({
    mutationFn: async (q: string) => (await items.whereIs(q)).data as any,
    onSuccess: (data) => setWhereAnswer(data?.answer ?? '未找到位置'),
  })

  const wherePhotoMutation = useMutation({
    mutationFn: async (file: File) => (await items.whereIsPhoto(file)).data as any,
    onSuccess: (data) => setWhereAnswer(data?.answer ?? '未找到位置'),
  })

  const expiring = useMemo(() => (expiringQuery.data ?? []).filter((item) => typeof item.days_until_expiry === 'number' && item.days_until_expiry <= 7), [expiringQuery.data])
  const docs = useMemo(() => (docsQuery.data ?? []).filter((item) => item.is_document), [docsQuery.data])
  const hasError = roomsQuery.isError || containersQuery.isError || itemsQuery.isError

  return (
    <div className="px-5 pt-6 pb-6 space-y-5">
      <div className="type-h1 text-nesio-ink">物品</div>

      {hasError && (
        <div className="ui-card p-4">
          <div className="type-body text-nesio-accent">物品页加载失败，请稍后重试。</div>
        </div>
      )}

      {(roomsQuery.isLoading || itemsQuery.isLoading) && (
        <div className="ui-card-plain p-4">
          <div className="type-body text-nesio-muted">正在加载物品数据...</div>
        </div>
      )}

      <div className="ui-card-plain px-4 py-3">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜索物品"
          className="w-full bg-transparent outline-none type-body text-nesio-ink"
        />
      </div>

      <div className="flex gap-2 overflow-x-auto scrollbar-hide">
        <button onClick={() => { setSelectedRoom(''); setSelectedContainer('') }} className={`ui-chip ${!selectedRoom ? 'ui-chip-active' : ''}`}>
          全部房间
        </button>
        {roomsQuery.data?.map((room) => (
          <button key={room.id} onClick={() => { setSelectedRoom(room.id); setSelectedContainer('') }} className={`ui-chip ${selectedRoom === room.id ? 'ui-chip-active' : ''}`}>
            {room.name}
          </button>
        ))}
      </div>

      <div className="flex gap-2 overflow-x-auto scrollbar-hide">
        <button onClick={() => setSelectedContainer('')} className={`ui-chip ${!selectedContainer ? 'ui-chip-active' : ''}`}>
          全部容器
        </button>
        {containersQuery.data?.map((container) => (
          <button key={container.id} onClick={() => setSelectedContainer(container.id)} className={`ui-chip ${selectedContainer === container.id ? 'ui-chip-active' : ''}`}>
            {container.name}
          </button>
        ))}
      </div>

      <div className="space-y-3">
        {!itemsQuery.isLoading && (itemsQuery.data ?? []).length === 0 && (
          <div className="ui-card-plain p-4 type-body text-nesio-muted">还没有物品记录，先用拍照或上传添加一个。</div>
        )}
        {(itemsQuery.data ?? []).map((item) => (
          <div key={item.id} className="ui-card-plain p-4">
            <div className="type-title text-nesio-ink">{item.name}</div>
            <div className="type-body text-nesio-muted mt-1">{item.room_name ?? '未设置房间'} · {item.container_name ?? '未设置容器'}</div>
            {typeof item.days_until_expiry === 'number' && (
              <div className="type-caption text-nesio-accent mt-2">{item.days_until_expiry < 0 ? `已过期 ${-item.days_until_expiry} 天` : `${item.days_until_expiry} 天后到期`}</div>
            )}
          </div>
        ))}
      </div>

      <div className="ui-card p-4 space-y-3">
        <div className="type-title text-nesio-ink">拍照告诉我放在哪</div>
        <div className="flex gap-2">
          <input value={whereQuery} onChange={(e) => setWhereQuery(e.target.value)} placeholder="输入物品名" className="ui-input flex-1" />
          <button onClick={() => whereMutation.mutate(whereQuery)} className="ui-btn-primary">查询</button>
        </div>
        <input ref={wherePhotoInputRef} type="file" accept="image/*" capture="environment" className="hidden" onChange={(e) => {
          const f = e.currentTarget.files?.[0]
          if (f) {
            wherePhotoMutation.mutate(f)
          }
          e.currentTarget.value = ''
        }} />
        <button onClick={() => wherePhotoInputRef.current?.click()} className="ui-btn-ghost">拍照查询</button>
        {whereAnswer && <div className="type-body text-nesio-muted">{whereAnswer}</div>}
      </div>

      <div className="ui-card-plain p-4">
        <div className="type-title text-nesio-ink mb-2">即将到期</div>
        {expiring.slice(0, 4).map((item) => (
          <div key={item.id} className="type-body text-nesio-muted py-1">{item.name}</div>
        ))}
      </div>

      <div className="ui-card-plain p-4">
        <div className="type-title text-nesio-ink mb-2">证件到期</div>
        {docs.slice(0, 4).map((item) => (
          <div key={item.id} className="type-body text-nesio-muted py-1">{item.name}</div>
        ))}
      </div>
    </div>
  )
}
