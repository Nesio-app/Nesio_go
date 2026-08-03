interface RoomOption {
  id: string
  name: string
}

interface ContainerOption {
  id: string
  name: string
}

interface Props {
  roomId: string
  containerId: string
  rooms: RoomOption[]
  containers: ContainerOption[]
  onRoomChange: (roomId: string) => void
  onContainerChange: (containerId: string) => void
}

export default function LocationPicker({
  roomId,
  containerId,
  rooms,
  containers,
  onRoomChange,
  onContainerChange,
}: Props) {
  return (
    <div className="space-y-3">
      <div className="grid grid-cols-[84px,1fr] items-center gap-3">
        <span className="type-body text-nesio-muted">存放位置</span>
        <select
          value={roomId}
          onChange={(e) => onRoomChange(e.target.value)}
          className="ui-input font-semibold bg-nesio-accentSoft"
        >
          <option value="">选择地点...</option>
          {rooms.map((room) => (
            <option key={room.id} value={room.id}>{room.name}</option>
          ))}
        </select>
      </div>

      {roomId && (
        <div className="grid grid-cols-[84px,1fr] items-center gap-3">
          <span className="type-body text-nesio-muted">容器</span>
          <select
            value={containerId}
            onChange={(e) => onContainerChange(e.target.value)}
            className="ui-input"
          >
            <option value="">未设置容器</option>
            {containers.map((container) => (
              <option key={container.id} value={container.id}>{container.name}</option>
            ))}
          </select>
        </div>
      )}
    </div>
  )
}
