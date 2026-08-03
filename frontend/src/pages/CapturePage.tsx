import { useEffect, useMemo, useRef, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { items } from '../api/client'
import CameraSheet from '../components/CameraSheet'
import CaptureBar from '../components/CaptureBar'

interface AnalyzeResult {
  extraction: Record<string, any>
  duplicates: Array<Record<string, any>>
  visual_hash: string
}

interface Props {
  onClose: () => void
  captureRequestToken?: number
  onAnalyzed: (payload: { file: File; previewUrl: string; result: AnalyzeResult }) => void
}

export default function CapturePage({ onClose, captureRequestToken, onAnalyzed }: Props) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [previewUrl, setPreviewUrl] = useState<string>('')

  const analyzeMutation = useMutation({
    mutationFn: async (source: File) => {
      const response = await items.analyze(source)
      return response.data as AnalyzeResult
    },
    onSuccess: (result) => {
      if (!file || !previewUrl) {
        return
      }
      onAnalyzed({ file, previewUrl, result })
    },
  })

  const filename = useMemo(() => file?.name ?? '', [file])

  const pickPhoto = () => {
    fileInputRef.current?.click()
  }

  useEffect(() => {
    if (!file && captureRequestToken) {
      fileInputRef.current?.click()
    }
  }, [captureRequestToken, file])

  const resetFile = () => {
    setFile(null)
    setPreviewUrl('')
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  const onFileSelected = (nextFile: File | null) => {
    if (!nextFile) {
      return
    }
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl)
    }
    const url = URL.createObjectURL(nextFile)
    setFile(nextFile)
    setPreviewUrl(url)
  }

  const directSave = async () => {
    if (!file) {
      return
    }
    const defaultName = filename.replace(/\.[^/.]+$/, '') || '新物品'
    await items.create({ name: defaultName, tags: ['拍照', '直接保存'] })
    onClose()
  }

  return (
    <div className="h-full bg-nesio-ink text-white flex flex-col">
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        capture="environment"
        className="hidden"
        onChange={(event) => onFileSelected(event.currentTarget.files?.[0] ?? null)}
      />

      <div className="px-5 pt-5 pb-2 text-right">
        <button onClick={onClose} className="ui-btn ui-btn-secondary">关闭</button>
      </div>

      {!file && (
        <CameraSheet onPickPhoto={pickPhoto} />
      )}

      {file && (
        <>
          <div className="px-5 pt-2">
            <div className="mx-auto rounded-3xl bg-gradient-to-b from-white/20 to-white/5 px-6 py-6 text-center type-h2 font-bold leading-tight">
              圈住区域让 AI
              <br />
              识别;或直接存,
              <br />
              自己填名字
            </div>
          </div>

          <div className="relative mt-4 flex-1 overflow-hidden px-4">
            <img src={previewUrl} alt="preview" className="h-full w-full rounded-2xl object-cover" />
            <div className="pointer-events-none absolute inset-8 rounded-3xl border-4 border-nesio-accent/80" />
          </div>

          <CaptureBar
            onDirectSave={directSave}
            onAnalyze={() => file && analyzeMutation.mutate(file)}
            onRetake={resetFile}
            analyzing={analyzeMutation.isPending}
            disabled={!file}
          />
        </>
      )}
    </div>
  )
}
