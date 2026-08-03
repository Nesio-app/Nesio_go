interface Props {
  onPickPhoto: () => void
}

export default function CameraSheet({ onPickPhoto }: Props) {
  return (
    <div className="flex-1 flex items-center justify-center px-6">
      <button onClick={onPickPhoto} className="ui-btn w-full max-w-xs border border-white/30 bg-white/10 text-white">
        拍照识别物品
      </button>
    </div>
  )
}
