import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { containers, items, rooms } from '../api/client'
import DuplicateAlert from '../components/DuplicateAlert'
import LocationPicker from '../components/LocationPicker'

interface AnalyzeResult {
  extraction: Record<string, any>
  duplicates: Array<Record<string, any>>
  visual_hash: string
}

interface Props {
  previewUrl: string
  result: AnalyzeResult
  onClose: () => void
}

export default function RecognitionResultPage({ previewUrl, result, onClose }: Props) {
  const [name, setName] = useState(String(result.extraction?.name ?? ''))
  const [description, setDescription] = useState('')
  const [roomId, setRoomId] = useState<string>('')
  const [containerId, setContainerId] = useState<string>('')
  const [price, setPrice] = useState('')
  const [expiryDate, setExpiryDate] = useState('')
  const [reminderLabel, setReminderLabel] = useState('服药提醒')
  const [shouldCreateNew, setShouldCreateNew] = useState(result.duplicates.length === 0)

  const roomsQuery = useQuery({
    queryKey: ['rooms'],
    queryFn: async () => (await rooms.list()).data as Array<{ id: string; name: string; icon: string }>,
  })
  const containersQuery = useQuery({
    queryKey: ['containers', roomId],
    queryFn: async () => (await containers.list(roomId || undefined)).data as Array<{ id: string; name: string; icon: string }>,
  })

  const locationPlaceholder = useMemo(() => {
    const room = roomsQuery.data?.find((item) => item.id === roomId)
    const container = containersQuery.data?.find((item) => item.id === containerId)
    if (!room && !container) {
      return '选择地点...'
    }
    if (room && !container) {
      return room.name
    }
    return `${room?.name ?? ''} · ${container?.name ?? ''}`.trim()
  }, [roomId, containerId, roomsQuery.data, containersQuery.data])

  const createPayload = useMemo(() => {
    const extraction = result.extraction ?? {}
    const isDocument = Boolean(extraction.is_document)
    const documentType = typeof extraction.document_type === 'string' ? extraction.document_type : undefined
    const documentNumber = typeof extraction.document_number === 'string' ? extraction.document_number : undefined

    return {
      name: name || '未命名物品',
      body: description || undefined,
      room_id: roomId || undefined,
      container_id: containerId || undefined,
      expiry_date: expiryDate || undefined,
      quantity: 1,
      unit: '个',
      visual_hash: result.visual_hash,
      reminder_label: reminderLabel || undefined,
      tags: Array.isArray(extraction.tags) ? extraction.tags : ['拍照识别'],
      primary_image_url: previewUrl,
      is_document: isDocument,
      document_type: documentType,
      document_number: documentNumber,
    }
  }, [containerId, description, expiryDate, name, previewUrl, reminderLabel, result.extraction, result.visual_hash, roomId])

  const saveMutation = useMutation({
    mutationFn: async () => {
      await items.create(createPayload)
    },
    onSuccess: () => onClose(),
  })

  const mergeDuplicateMutation = useMutation({
    mutationFn: async (targetItemId: string) => {
      await items.create(createPayload).then((res) => res.data as { id: string })
        .then(async (created) => {
          await items.duplicate(created.id, targetItemId, 1)
        })
    },
    onSuccess: () => onClose(),
  })

  return (
    <div className="h-full bg-nesio-bg text-nesio-ink overflow-y-auto">
      <div className="bg-white px-6 pt-6 pb-5 sticky top-0 z-20 border-b border-nesio-border">
        <div className="flex items-center justify-between">
          <button onClick={onClose} className="ui-icon-btn h-10 w-10 rounded-full text-2xl leading-none text-nesio-muted">×</button>
          <h1 className="type-h1">识别结果</h1>
          <div className="w-10" />
        </div>
      </div>

      <img src={previewUrl} alt="analyzed" className="h-72 w-full object-cover" />

      <div className="px-6 py-5 space-y-5">
        <section className="ui-card p-5 space-y-4">
          <div className="rounded-xl border border-nesio-border bg-nesio-bg px-4 py-3 flex items-center justify-between gap-3">
            <input value={name} onChange={(e) => setName(e.target.value)} className="w-full bg-transparent type-h2 font-bold outline-none" />
            <button onClick={() => setName('')} className="text-xl text-nesio-muted">×</button>
          </div>

          <input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="补充一句描述...（可选）"
            className="w-full border-b border-nesio-border bg-transparent px-2 py-3 type-body text-nesio-muted outline-none"
          />

          <LocationPicker
            roomId={roomId}
            containerId={containerId}
            rooms={(roomsQuery.data ?? []).map((room) => ({ id: room.id, name: room.name }))}
            containers={(containersQuery.data ?? []).map((container) => ({ id: container.id, name: container.name }))}
            onRoomChange={(value) => {
              setRoomId(value)
              setContainerId('')
            }}
            onContainerChange={setContainerId}
          />

          <div className="grid grid-cols-[84px,1fr] items-center gap-3">
            <span className="type-body text-nesio-muted">价格</span>
            <input value={price} onChange={(e) => setPrice(e.target.value)} placeholder="如 24.99" className="ui-input" />
          </div>

          <div className="grid grid-cols-[84px,1fr] items-center gap-3">
            <span className="type-body text-nesio-muted">有效期</span>
            <input type="date" value={expiryDate} onChange={(e) => setExpiryDate(e.target.value)} className="ui-input" />
          </div>
        </section>

        <section className="ui-card p-5 space-y-3">
          <div className="rounded-xl border border-nesio-border bg-nesio-bg px-4 py-3 flex items-center justify-between gap-3">
            <input value={reminderLabel} onChange={(e) => setReminderLabel(e.target.value)} className="w-full bg-transparent type-h2 font-bold outline-none" />
            <button onClick={() => setReminderLabel('')} className="text-xl text-nesio-muted">×</button>
          </div>
          <div className="type-body text-nesio-muted">建议地点: {locationPlaceholder}</div>
          {result.duplicates.length > 0 && !shouldCreateNew && (
            <DuplicateAlert
              duplicates={result.duplicates}
              onUpdate={(targetItemId) => mergeDuplicateMutation.mutate(targetItemId)}
              onCreateNew={() => setShouldCreateNew(true)}
              onCancel={onClose}
              isPending={mergeDuplicateMutation.isPending}
            />
          )}
        </section>

        <button
          onClick={() => saveMutation.mutate()}
          disabled={saveMutation.isPending || mergeDuplicateMutation.isPending || (!shouldCreateNew && result.duplicates.length > 0)}
          className="ui-btn-primary w-full"
        >
          {saveMutation.isPending ? '保存中...' : shouldCreateNew ? '保存物品' : '请选择“更新现有记录”或“这是新的”'}
        </button>
      </div>
    </div>
  )
}
