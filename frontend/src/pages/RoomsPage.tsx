import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { IconArrowLeft } from '../icons'
import { items, rooms } from '../api/client'

interface Props {
  onBack: () => void
}

export default function RoomsPage({ onBack }: Props) {
  const queryClient = useQueryClient()
  const [newRoomName, setNewRoomName] = useState('')
  const [newRoomIcon, setNewRoomIcon] = useState('')

  const roomsQuery = useQuery({
    queryKey: ['rooms'],
    queryFn: async () => (await rooms.list()).data as Array<{ id: string; name: string; icon?: string | null }>,
  })

  const itemsQuery = useQuery({
    queryKey: ['items-room-summary'],
    queryFn: async () => (await items.list()).data as Array<{ id: string; room_name?: string | null }>,
  })

  const roomCounts = useMemo(() => {
    const map = new Map<string, number>()
    for (const item of itemsQuery.data ?? []) {
      const key = item.room_name?.trim()
      if (!key) {
        continue
      }
      map.set(key, (map.get(key) ?? 0) + 1)
    }
    return map
  }, [itemsQuery.data])

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
        queryClient.invalidateQueries({ queryKey: ['items-room-summary'] }),
      ])
    },
  })

  const deleteRoomMutation = useMutation({
    mutationFn: async (roomId: string) => {
      await rooms.remove(roomId)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['rooms'] }),
        queryClient.invalidateQueries({ queryKey: ['items'] }),
        queryClient.invalidateQueries({ queryKey: ['items-room-summary'] }),
      ])
    },
  })

  return (
    <div className="px-5 pt-6 pb-6 space-y-5">
      <div className="flex items-center justify-between">
        <button onClick={onBack} className="ui-icon-btn w-10 h-10 rounded-full bg-nesio-accentSoft text-nesio-accent">
          <IconArrowLeft className="w-8 h-8" />
        </button>
        <div className="type-h1 text-nesio-ink">房间管理</div>
        <div className="w-10 h-10" />
      </div>

      <div className="ui-card p-4 space-y-3">
        <div className="type-title text-nesio-ink">新增房间</div>
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
            新建
          </button>
        </div>
      </div>

      <div className="space-y-3">
        {(roomsQuery.data ?? []).map((room) => (
          <div key={room.id} className="ui-card-plain p-4 flex items-center justify-between gap-3">
            <div>
              <div className="type-title text-nesio-ink">{room.name}</div>
              <div className="type-caption text-nesio-muted mt-1">物品 {roomCounts.get(room.name) ?? 0} 件</div>
            </div>
            <button
              onClick={() => deleteRoomMutation.mutate(room.id)}
              disabled={deleteRoomMutation.isPending}
              className="ui-btn-ghost"
            >
              删除
            </button>
          </div>
        ))}

        {(roomsQuery.data ?? []).length === 0 && (
          <div className="ui-card-plain p-4 type-body text-nesio-muted">还没有房间，先创建一个。</div>
        )}
      </div>
    </div>
  )
}
