import { useEffect, useMemo, useRef, useState, type PointerEvent } from 'react'
import { useMutation } from '@tanstack/react-query'
import { items } from '../api/client'
import CameraSheet from '../components/CameraSheet'
import CaptureBar from '../components/CaptureBar'

interface CropRect {
  x: number
  y: number
  width: number
  height: number
}

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
  const imageRef = useRef<HTMLImageElement>(null)
  const overlayRef = useRef<HTMLDivElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [previewUrl, setPreviewUrl] = useState<string>('')
  const [selection, setSelection] = useState<CropRect | null>(null)
  const [dragStart, setDragStart] = useState<{ x: number; y: number } | null>(null)
  const [isDragging, setIsDragging] = useState(false)

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
    setSelection(null)
    setDragStart(null)
    setIsDragging(false)
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
    setSelection(null)
    setDragStart(null)
    setIsDragging(false)
  }

  const getRelativePoint = (clientX: number, clientY: number) => {
    const overlay = overlayRef.current
    if (!overlay) {
      return null
    }
    const rect = overlay.getBoundingClientRect()
    const x = Math.max(0, Math.min(rect.width, clientX - rect.left))
    const y = Math.max(0, Math.min(rect.height, clientY - rect.top))
    return { x, y }
  }

  const handlePointerDown = (event: PointerEvent<HTMLDivElement>) => {
    const point = getRelativePoint(event.clientX, event.clientY)
    if (!point) {
      return
    }
    setDragStart(point)
    setSelection({ x: point.x, y: point.y, width: 0, height: 0 })
    setIsDragging(true)
    event.currentTarget.setPointerCapture(event.pointerId)
  }

  const handlePointerMove = (event: PointerEvent<HTMLDivElement>) => {
    if (!isDragging || !dragStart) {
      return
    }
    const point = getRelativePoint(event.clientX, event.clientY)
    if (!point) {
      return
    }
    const x = Math.min(dragStart.x, point.x)
    const y = Math.min(dragStart.y, point.y)
    const width = Math.abs(point.x - dragStart.x)
    const height = Math.abs(point.y - dragStart.y)
    setSelection({ x, y, width, height })
  }

  const handlePointerUp = (event: PointerEvent<HTMLDivElement>) => {
    if (!isDragging) {
      return
    }
    setIsDragging(false)
    setDragStart(null)
    if (selection && (selection.width < 24 || selection.height < 24)) {
      setSelection(null)
    }
    event.currentTarget.releasePointerCapture(event.pointerId)
  }

  const createCroppedFile = async (source: File, crop: CropRect) => {
    const image = imageRef.current
    if (!image) {
      return source
    }
    const displayWidth = image.clientWidth
    const displayHeight = image.clientHeight
    const naturalWidth = image.naturalWidth
    const naturalHeight = image.naturalHeight
    if (!displayWidth || !displayHeight || !naturalWidth || !naturalHeight) {
      return source
    }

    const scaleX = naturalWidth / displayWidth
    const scaleY = naturalHeight / displayHeight
    const sx = Math.max(0, Math.floor(crop.x * scaleX))
    const sy = Math.max(0, Math.floor(crop.y * scaleY))
    const sw = Math.max(1, Math.floor(crop.width * scaleX))
    const sh = Math.max(1, Math.floor(crop.height * scaleY))

    const canvas = document.createElement('canvas')
    canvas.width = sw
    canvas.height = sh
    const ctx = canvas.getContext('2d')
    if (!ctx) {
      return source
    }

    ctx.drawImage(image, sx, sy, sw, sh, 0, 0, sw, sh)
    const mime = source.type || 'image/jpeg'
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, mime, 0.92))
    if (!blob) {
      return source
    }
    const ext = mime.includes('png') ? 'png' : 'jpg'
    const baseName = source.name.replace(/\.[^/.]+$/, '')
    return new File([blob], `${baseName}_crop.${ext}`, { type: mime })
  }

  const runAnalyze = async () => {
    if (!file) {
      return
    }
    let source = file
    if (selection && selection.width >= 24 && selection.height >= 24) {
      source = await createCroppedFile(file, selection)
    }
    analyzeMutation.mutate(source)
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
            <div className="relative h-full w-full">
              <img ref={imageRef} src={previewUrl} alt="preview" className="h-full w-full rounded-2xl object-contain bg-black/20" />
              <div
                ref={overlayRef}
                className="absolute inset-0 rounded-2xl"
                onPointerDown={handlePointerDown}
                onPointerMove={handlePointerMove}
                onPointerUp={handlePointerUp}
              >
                {selection && (
                  <div
                    className="absolute border-2 border-rose-300 bg-rose-300/15 rounded-xl"
                    style={{
                      left: selection.x,
                      top: selection.y,
                      width: selection.width,
                      height: selection.height,
                    }}
                  />
                )}
              </div>
            </div>
          </div>

          <CaptureBar
            onDirectSave={directSave}
            onAnalyze={runAnalyze}
            onRetake={resetFile}
            analyzing={analyzeMutation.isPending}
            disabled={!file}
          />
        </>
      )}
    </div>
  )
}
