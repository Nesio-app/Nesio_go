interface Props {
  onDirectSave: () => void
  onAnalyze: () => void
  onRetake: () => void
  analyzing: boolean
  saving?: boolean
  disabled?: boolean
}

export default function CaptureBar({ onDirectSave, onAnalyze, onRetake, analyzing, saving = false, disabled }: Props) {
  return (
    <div className="px-5 pb-7 pt-5 space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <button
          type="button"
          onClick={onDirectSave}
          disabled={disabled || analyzing || saving}
          className="ui-btn border border-white/45 bg-white/20 text-white"
        >
          {saving ? '保存中...' : '直接存'}
        </button>
        <button
          type="button"
          onClick={onAnalyze}
          disabled={disabled || analyzing || saving}
          className="ui-btn border border-white/45 bg-white/20 text-white"
        >
          {analyzing ? '识别中...' : 'AI识别选区'}
        </button>
      </div>
      <button type="button" onClick={onRetake} disabled={analyzing || saving} className="ui-btn mx-auto border border-white/35 bg-white/15 px-8 text-white">
        ↩ 重拍
      </button>
    </div>
  )
}
