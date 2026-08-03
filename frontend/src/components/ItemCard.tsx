interface ItemCardData {
  id: string
  name: string
  room_name?: string | null
  container_name?: string | null
  days_until_expiry?: number | null
}

interface Props {
  item: ItemCardData
  onOpen?: (itemId: string) => void
}

export default function ItemCard({ item, onOpen }: Props) {
  return (
    <button
      onClick={() => onOpen?.(item.id)}
      className="ui-card-plain p-4 text-left w-full"
    >
      <div className="type-title text-nesio-ink">{item.name}</div>
      <div className="type-body text-nesio-muted mt-1">{item.room_name ?? '未设置房间'} · {item.container_name ?? '未设置容器'}</div>
      {typeof item.days_until_expiry === 'number' && (
        <div className="type-caption text-nesio-accent mt-2">
          {item.days_until_expiry < 0 ? `已过期 ${-item.days_until_expiry} 天` : `${item.days_until_expiry} 天后到期`}
        </div>
      )}
    </button>
  )
}
