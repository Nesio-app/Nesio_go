import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { containers, items, rooms } from '../api/client'
import WhereIs from '../components/WhereIs'

interface Props {
  onOpenItem?: (itemId: string) => void
}

export default function ItemsPage({ onOpenItem }: Props) {
  const queryClient = useQueryClient()
  const [selectedRoom, setSelectedRoom] = useState<string>('')
  const [selectedContainer, setSelectedContainer] = useState<string>('')
  const [query, setQuery] = useState('')
  const [newRoomName, setNewRoomName] = useState('')
  const [newRoomIcon, setNewRoomIcon] = useState('')
  const [newContainerName, setNewContainerName] = useState('')
  const [newContainerIcon, setNewContainerIcon] = useState('')

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

  const expiring = useMemo(() => (expiringQuery.data ?? []).filter((item) => typeof item.days_until_expiry === 'number' && item.days_until_expiry <= 7), [expiringQuery.data])
  const docs = useMemo(() => (docsQuery.data ?? []).filter((item) => item.is_document), [docsQuery.data])
  const hasError = roomsQuery.isError || containersQuery.isError || itemsQuery.isError

  const createRoomMutation = useMutation({
    mutationFn: async () => {
      await rooms.create({
        name: newRoomName.trim(),
        icon: newRoomIcon.trim() || undefined,
      })
    },
    onSuccess: async () => {
      setNewRoomName('')
      setNewRoomIcon('')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['rooms'] }),
        queryClient.invalidateQueries({ queryKey: ['items'] }),
      ])
    },
  })

  const createContainerMutation = useMutation({
    mutationFn: async () => {
      await containers.create({
        name: newContainerName.trim(),
        icon: newContainerIcon.trim() || undefined,
        room_id: selectedRoom || undefined,
      })
    },
    onSuccess: async () => {
      setNewContainerName('')
      setNewContainerIcon('')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['containers'] }),
        queryClient.invalidateQueries({ queryKey: ['items'] }),
      ])
    },
  })

  const deleteRoomMutation = useMutation({
    mutationFn: async (roomId: string) => {
      await rooms.remove(roomId)
    },
    onSuccess: async () => {
      setSelectedRoom('')
      setSelectedContainer('')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['rooms'] }),
        queryClient.invalidateQueries({ queryKey: ['containers'] }),
        queryClient.invalidateQueries({ queryKey: ['items'] }),
      ])
    },
  })

  const deleteContainerMutation = useMutation({
    mutationFn: async (containerId: string) => {
      await containers.remove(containerId)
    },
    onSuccess: async () => {
      setSelectedContainer('')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['containers'] }),
        queryClient.invalidateQueries({ queryKey: ['items'] }),
      ])
    },
  })

  const snoozeExpiryMutation = useMutation({
    mutationFn: async (itemId: string) => {
      await items.snoozeExpiry(itemId)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['items-expiring'] }),
        queryClient.invalidateQueries({ queryKey: ['items'] }),
      ])
    },
  })

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

      <div className="ui-card p-4 space-y-3">
        <div className="type-title text-nesio-ink">空间管理</div>
        <div className="grid grid-cols-[1fr,92px,88px] gap-2">
          <input
            value={newRoomName}
            onChange={(e) => setNewRoomName(e.target.value)}
            placeholder="新房间名（如：客厅）"
            className="ui-input"
          />
          <input
            value={newRoomIcon}
            onChange={(e) => setNewRoomIcon(e.target.value)}
            placeholder="图标"
            className="ui-input"
          />
          <button
            onClick={() => createRoomMutation.mutate()}
            disabled={createRoomMutation.isPending || !newRoomName.trim()}
            className="ui-btn-secondary"
          >
            新建房间
          </button>
        </div>

        <div className="grid grid-cols-[1fr,92px,88px] gap-2">
          <input
            value={newContainerName}
            onChange={(e) => setNewContainerName(e.target.value)}
            placeholder={selectedRoom ? '新容器名（会绑定当前房间）' : '新容器名（不绑定房间）'}
            className="ui-input"
          />
          <input
            value={newContainerIcon}
            onChange={(e) => setNewContainerIcon(e.target.value)}
            placeholder="图标"
            className="ui-input"
          />
          <button
            onClick={() => createContainerMutation.mutate()}
            disabled={createContainerMutation.isPending || !newContainerName.trim()}
            className="ui-btn-secondary"
          >
            新建容器
          </button>
        </div>

        {selectedRoom && (
          <button
            onClick={() => deleteRoomMutation.mutate(selectedRoom)}
            disabled={deleteRoomMutation.isPending}
            className="ui-btn-ghost"
          >
            {deleteRoomMutation.isPending ? '删除房间中...' : '删除当前房间'}
          </button>
        )}
        {selectedContainer && (
          <button
            onClick={() => deleteContainerMutation.mutate(selectedContainer)}
            disabled={deleteContainerMutation.isPending}
            className="ui-btn-ghost"
          >
            {deleteContainerMutation.isPending ? '删除容器中...' : '删除当前容器'}
          </button>
        )}
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
          <button
            key={item.id}
            onClick={() => onOpenItem?.(item.id)}
            className="ui-card-plain p-4 text-left w-full"
          >
            <div className="type-title text-nesio-ink">{item.name}</div>
            <div className="type-body text-nesio-muted mt-1">{item.room_name ?? '未设置房间'} · {item.container_name ?? '未设置容器'}</div>
            {typeof item.days_until_expiry === 'number' && (
              <div className="type-caption text-nesio-accent mt-2">{item.days_until_expiry < 0 ? `已过期 ${-item.days_until_expiry} 天` : `${item.days_until_expiry} 天后到期`}</div>
            )}
          </button>
        ))}
      </div>

      <WhereIs onOpenItem={onOpenItem} />

      <div className="ui-card-plain p-4">
        <div className="type-title text-nesio-ink mb-2">即将到期</div>
        {expiring.slice(0, 4).map((item) => (
          <div key={item.id} className="py-2 border-b border-nesio-border last:border-b-0">
            <button
              onClick={() => onOpenItem?.(item.id)}
              className="type-body text-nesio-muted text-left w-full"
            >
              {item.name}
            </button>
            <div className="mt-1 flex justify-between items-center">
              <div className="type-caption text-nesio-muted">
                {typeof item.days_until_expiry === 'number'
                  ? (item.days_until_expiry >= 0 ? `${item.days_until_expiry} 天后到期` : `已过期 ${-item.days_until_expiry} 天`)
                  : '到期时间未知'}
              </div>
              <button
                onClick={() => snoozeExpiryMutation.mutate(item.id)}
                disabled={snoozeExpiryMutation.isPending}
                className="ui-btn-ghost px-2 py-1"
              >
                稍后提醒
              </button>
            </div>
          </div>
        ))}
        {expiring.length === 0 && <div className="type-body text-nesio-muted">暂无临期物品。</div>}
      </div>

      <div className="ui-card-plain p-4">
        <div className="type-title text-nesio-ink mb-2">证件到期</div>
        {docs.slice(0, 4).map((item) => (
          <button
            key={item.id}
            onClick={() => onOpenItem?.(item.id)}
            className="type-body text-nesio-muted py-1 text-left w-full"
          >
            {item.name}
          </button>
        ))}
        {docs.length === 0 && <div className="type-body text-nesio-muted">暂无证件物品。</div>}
      </div>
    </div>
  )
}
