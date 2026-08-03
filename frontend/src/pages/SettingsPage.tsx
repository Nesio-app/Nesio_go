import { useMutation, useQuery } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { Capacitor, registerPlugin } from '@capacitor/core'
import {
  IconSun, IconShield, IconGift, IconHelp, IconBulb, IconChevronRight,
  IconSettings, IconArrowLeft, IconMapPin, IconHeart, IconSparkles, IconActivity
} from '../icons'
import { connectors as connectorsApi, dailyBriefs, dataExport, gmail } from '../api/client'

interface VisionPlugin {
  recognizeText(options: { base64Image: string }): Promise<{ text: string; lines: string[] }>
  classifyImage(options: { base64Image: string }): Promise<{ labels: string[] }>
  inferSmartModel(options: { text: string }): Promise<{ intent: string; summary: string; confidence: number; input: string }>
}

interface LocationPlugin {
  requestPermission(): Promise<{ granted: boolean }>
  currentLocation(): Promise<{ latitude: number; longitude: number; accuracy: number }>
}

interface PushNotificationPlugin {
  requestPermission(): Promise<{ granted: boolean }>
  getDeviceToken(): Promise<{ token: string }>
}

interface HealthKitPlugin {
  requestPermission(): Promise<{ granted: boolean }>
  readTodaySteps(): Promise<{ steps: number }>
}

type NativePluginRegistry = {
  vision?: VisionPlugin
  location?: LocationPlugin
  push?: PushNotificationPlugin
  health?: HealthKitPlugin
}

const pluginRegistry = ((globalThis as any).__nesioNativePlugins ??= {}) as NativePluginRegistry
const visionPlugin = (pluginRegistry.vision ??= registerPlugin<VisionPlugin>('VisionPlugin'))
const locationPlugin = (pluginRegistry.location ??= registerPlugin<LocationPlugin>('LocationPlugin'))
const pushNotificationPlugin = (pluginRegistry.push ??= registerPlugin<PushNotificationPlugin>('PushNotification'))
const healthKitPlugin = (pluginRegistry.health ??= registerPlugin<HealthKitPlugin>('HealthKitPlugin'))

const menuItems = [
  { icon: IconSun, label: '外观与语言' },
  { icon: IconShield, label: '数据与隐私' },
  { icon: IconGift, label: '会员 · Pro' },
  { icon: IconHelp, label: '帮助与反馈' },
  { icon: IconBulb, label: 'Lab' },
]

interface Props {
  onBack: () => void
  themeMode: 'light' | 'dark'
  palette: 'dawn' | 'ocean' | 'forest'
  onThemeChange: (mode: 'light' | 'dark') => void
  onPaletteChange: (palette: 'dawn' | 'ocean' | 'forest') => void
}

export default function SettingsPage({ onBack, themeMode, palette, onThemeChange, onPaletteChange }: Props) {
  const nativeVisionInputRef = useRef<HTMLInputElement>(null)
  const [nativeMode, setNativeMode] = useState<'ocr' | 'vision' | null>(null)
  const [deviceStatus, setDeviceStatus] = useState('尚未检查设备能力')
  const [nativeEvidence, setNativeEvidence] = useState('')
  const connectorListQuery = useQuery({
    queryKey: ['connectors'],
    queryFn: async () => {
      const r = await connectorsApi.list()
      return r.data as Array<{ id: string; provider: string; is_active: boolean; last_sync_at?: string | null }>
    },
  })
  const connectorProvidersQuery = useQuery({
    queryKey: ['connector-providers'],
    queryFn: async () => {
      const r = await connectorsApi.providers()
      return r.data as Array<{ provider: string; label: string; status: string }>
    },
  })
  const connectedProviderSet = new Set((connectorListQuery.data ?? []).filter((c) => c.is_active).map((c) => c.provider))

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const provider = params.get('connector')
    const status = params.get('status')
    if (!provider || !status) {
      return
    }

    if (status === 'connected') {
      alert(`${provider} 已连接。`)
    } else if (status === 'oauth_code_received') {
      alert(`${provider} 授权码已收到，但尚未完成 access token 交换。请配置对应连接器 token 交换流程。`)
    } else {
      alert(`${provider} 授权状态：${status}`)
    }

    params.delete('connector')
    params.delete('status')
    const next = params.toString()
    const nextURL = `${window.location.pathname}${next ? `?${next}` : ''}${window.location.hash}`
    window.history.replaceState({}, '', nextURL)
  }, [])

  const exportMutation = useMutation({
    mutationFn: async () => {
      const response = await dataExport.run()
      return response.data as Record<string, any>
    },
    onSuccess: (payload) => {
      const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `nesio-export-${new Date().toISOString().slice(0, 10)}.json`
      link.click()
      URL.revokeObjectURL(url)
    },
  })

  const markBriefReadMutation = useMutation({
    mutationFn: async () => {
      const brief = await dailyBriefs.byDay(new Date().toISOString().slice(0, 10))
      const id = (brief.data as { id?: string })?.id
      if (!id) {
        throw new Error('missing brief id')
      }
      await dailyBriefs.read(id)
    },
  })

  const connectGmail = async () => {
    try {
      const { data } = await gmail.authorizeUrl()
      window.location.href = data.auth_url as string
    } catch (error: any) {
      const message = error?.response?.data?.message || error?.response?.data?.error || 'Gmail OAuth 尚未配置：请在服务器设置 GOOGLE_CLIENT_ID 和 GOOGLE_CLIENT_SECRET。'
      alert(message)
    }
  }

  const fileToBase64 = (file: File) => new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const value = reader.result
      if (typeof value === 'string') {
        resolve(value.split(',')[1] ?? '')
      } else {
        reject(new Error('unable to read file'))
      }
    }
    reader.onerror = () => reject(new Error('unable to read file'))
    reader.readAsDataURL(file)
  })

  const runNativeVisionOCR = async (file: File) => {
    if (!Capacitor.isNativePlatform()) {
      setDeviceStatus('本机 OCR 仅在 iOS 原生环境可用。')
      return
    }
    const base64 = await fileToBase64(file)
    const result = await visionPlugin.recognizeText({ base64Image: base64 })
    const text = result.text.trim() || '未识别到文本'
    setNativeEvidence(result.text.trim())
    setDeviceStatus(text)
  }

  const runNativeVision = async (file: File) => {
    if (!Capacitor.isNativePlatform()) {
      setDeviceStatus('本机 Vision 仅在 iOS 原生环境可用。')
      return
    }
    const base64 = await fileToBase64(file)
    const result = await visionPlugin.classifyImage({ base64Image: base64 })
    const labels = result.labels.length > 0 ? result.labels.join('，') : '未识别到图像标签'
    setNativeEvidence(result.labels.join('\n'))
    setDeviceStatus(labels)
  }

  const runNativeSmartModel = async () => {
    if (!Capacitor.isNativePlatform()) {
      setDeviceStatus('本机智能模型仅在 iOS 原生环境可用。')
      return
    }
    const text = nativeEvidence.trim()
    if (!text) {
      setDeviceStatus('请先运行本机 OCR 或本机 Vision，再做本机智能推理。')
      return
    }
    const result = await visionPlugin.inferSmartModel({ text })
    setDeviceStatus(`${result.summary}（${result.intent} · ${Math.round(result.confidence * 100)}%）`)
  }

  const requestLocation = async () => {
    if (!Capacitor.isNativePlatform()) {
      setDeviceStatus('定位能力仅在 iOS 原生环境可用。')
      return
    }
    const permission = await locationPlugin.requestPermission()
    if (!permission.granted) {
      setDeviceStatus('定位权限未授权。')
      return
    }
    const location = await locationPlugin.currentLocation()
    setDeviceStatus(`当前位置 ${location.latitude.toFixed(5)}, ${location.longitude.toFixed(5)} · 精度 ${Math.round(location.accuracy)}m`)
  }

  const requestPushPermission = async () => {
    if (!Capacitor.isNativePlatform()) {
      setDeviceStatus('推送能力仅在 iOS 原生环境可用。')
      return
    }
    const permission = await pushNotificationPlugin.requestPermission()
    if (!permission.granted) {
      setDeviceStatus('推送权限未授权。')
      return
    }
    const token = await pushNotificationPlugin.getDeviceToken()
    setDeviceStatus(token.token ? `推送令牌：${token.token}` : '已授权，但当前没有设备令牌。')
  }

  const readHealthSteps = async () => {
    if (!Capacitor.isNativePlatform()) {
      setDeviceStatus('健康数据仅在 iOS 原生环境可用。')
      return
    }
    const permission = await healthKitPlugin.requestPermission()
    if (!permission.granted) {
      setDeviceStatus('健康权限未授权。')
      return
    }
    const steps = await healthKitPlugin.readTodaySteps()
    setDeviceStatus(`今日步数 ${steps.steps}`)
  }

  return (
    <div className="px-5 pt-6 pb-4 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <button
          onClick={onBack}
          className="w-8 h-8 flex items-center justify-center text-nesio-ink active:scale-90 transition"
        >
          <IconArrowLeft className="w-5 h-5" />
        </button>
        <button className="w-8 h-8 flex items-center justify-center text-nesio-muted active:scale-90 transition">
          <IconSettings className="w-5 h-5" />
        </button>
      </div>

      {/* Profile Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-14 h-14 rounded-full bg-nesio-accentSoft overflow-hidden flex items-center justify-center">
            <span className="text-xl font-bold text-nesio-accent">婧</span>
          </div>
          <span className="text-xl font-bold text-nesio-ink">婧</span>
          <IconChevronRight className="w-5 h-5 text-nesio-muted" />
        </div>
        <button
          onClick={onBack}
          className="text-nesio-accent text-sm font-medium active:opacity-70 transition"
        >
          返回今天
        </button>
      </div>

      {/* Connectors */}
      <div className="nesio-card p-4 space-y-3">
        <div className="text-sm font-semibold text-nesio-ink">连接器</div>
        {(connectorProvidersQuery.data ?? []).map((provider) => {
          const isConnected = connectedProviderSet.has(provider.provider)
          const canOAuth = ['gmail', 'tesla_fleet', 'granola', 'plaid', 'flomo', 'google_timeline', 'apple_health'].includes(provider.provider)
          return (
            <div key={provider.provider} className="flex items-center justify-between gap-3 rounded-2xl bg-white/70 px-3 py-3">
              <div>
                <div className="text-sm font-medium text-nesio-ink">{provider.label}</div>
                <div className="text-xs text-nesio-muted">
                  {isConnected ? '已连接' : '未连接'} · {provider.status}
                </div>
              </div>
              <button
                onClick={() => {
                  if (canOAuth) {
                    if (provider.provider === 'gmail') {
                      void connectGmail()
                      return
                    }
                    void connectorsApi.auth(provider.provider).then(({ data }) => {
                      window.location.href = data.auth_url as string
                    }).catch((error: any) => {
                      const message = error?.response?.data?.message || error?.response?.data?.error || '连接失败，请稍后重试。'
                      alert(message)
                    })
                    return
                  }
                }}
                className={`ui-btn rounded-full px-4 ${
                  isConnected ? 'bg-nesio-border text-nesio-muted' : 'bg-nesio-accent text-white'
                }`}
              >
                {isConnected ? '重新连接' : '连接'}
              </button>
            </div>
          )
        })}
        {(connectorProvidersQuery.data ?? []).length === 0 && (
          <div className="text-sm text-nesio-muted">正在加载连接器列表...</div>
        )}
      </div>

      <div className="nesio-card p-4 space-y-3">
        <div className="text-sm font-semibold text-nesio-ink">设备能力</div>
        <div className="grid grid-cols-1 gap-2">
          <button className="ui-btn-secondary w-full flex items-center justify-center gap-2" onClick={() => {
            setNativeMode('ocr')
            nativeVisionInputRef.current?.click()
          }}>
            <IconSparkles className="w-4 h-4" />
            本机 OCR
          </button>
          <button className="ui-btn-secondary w-full flex items-center justify-center gap-2" onClick={() => {
            setNativeMode('vision')
            nativeVisionInputRef.current?.click()
          }}>
            <IconSparkles className="w-4 h-4" />
            本机 Vision
          </button>
          <button className="ui-btn-secondary w-full flex items-center justify-center gap-2" onClick={() => void runNativeSmartModel()}>
            <IconActivity className="w-4 h-4" />
            本机智能模型
          </button>
          <button className="ui-btn-secondary w-full flex items-center justify-center gap-2" onClick={() => void requestLocation()}>
            <IconMapPin className="w-4 h-4" />
            获取定位
          </button>
          <button className="ui-btn-secondary w-full flex items-center justify-center gap-2" onClick={() => void requestPushPermission()}>
            <IconActivity className="w-4 h-4" />
            推送权限
          </button>
          <button className="ui-btn-secondary w-full flex items-center justify-center gap-2" onClick={() => void readHealthSteps()}>
            <IconHeart className="w-4 h-4" />
            读取今日步数
          </button>
        </div>
        <div className="rounded-2xl bg-white/70 px-3 py-3 text-sm text-nesio-muted leading-relaxed">
          {deviceStatus}
        </div>
        <div className="text-xs text-nesio-muted">
          {nativeEvidence ? '已缓存本机识别结果，可直接触发本机智能模型。' : '先运行 OCR 或 Vision，再让本机智能模型总结。'}
        </div>
        <div className="text-xs text-nesio-muted">
          当前平台：{Capacitor.getPlatform()} {Capacitor.isNativePlatform() ? '· 原生可用' : '· Web 预览'}
        </div>
        <input
          ref={nativeVisionInputRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={(event) => {
            const file = event.target.files?.[0]
            event.target.value = ''
            if (!file) {
              return
            }
            const mode = nativeMode
            setNativeMode(null)
            if (mode === 'vision') {
              void runNativeVision(file)
              return
            }
            void runNativeVisionOCR(file)
          }}
        />
      </div>

      <div className="nesio-card p-4 space-y-4">
        <div>
          <div className="text-sm text-nesio-muted mb-2">主题模式</div>
          <div className="flex gap-2">
            {(['light', 'dark'] as const).map((mode) => (
              <button
                key={mode}
                onClick={() => onThemeChange(mode)}
                className={`ui-chip transition ${themeMode === mode ? 'ui-chip-active' : ''}`}
              >
                {mode === 'light' ? '浅色' : '深色'}
              </button>
            ))}
          </div>
        </div>
        <div>
          <div className="text-sm text-nesio-muted mb-2">配色</div>
          <div className="flex gap-2 flex-wrap">
            {([
              { key: 'dawn', label: '晨雾' },
              { key: 'ocean', label: '海盐' },
              { key: 'forest', label: '林地' },
            ] as const).map((item) => (
              <button
                key={item.key}
                onClick={() => onPaletteChange(item.key)}
                className={`ui-chip transition ${palette === item.key ? 'ui-chip-active' : ''}`}
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>

        <div className="grid grid-cols-1 gap-2">
          <button
            onClick={() => exportMutation.mutate()}
            disabled={exportMutation.isPending}
            className="ui-btn-secondary w-full"
          >
            {exportMutation.isPending ? '导出中...' : '导出全部数据'}
          </button>
          <button
            onClick={() => markBriefReadMutation.mutate()}
            disabled={markBriefReadMutation.isPending}
            className="ui-btn-ghost w-full"
          >
            {markBriefReadMutation.isPending ? '处理中...' : '标记今日日报已读'}
          </button>
        </div>
      </div>

      {/* Menu Items */}
      <div className="space-y-3">
        {menuItems.map((item) => (
          <button
            key={item.label}
            className="w-full ui-card-plain p-4 flex items-center gap-4 active:scale-[0.99] transition"
          >
            <div className="w-12 h-12 rounded-2xl flex items-center justify-center bg-nesio-accentSoft text-nesio-accent">
              <item.icon className="w-6 h-6" />
            </div>
            <span className="flex-1 text-left text-base font-medium text-nesio-ink">{item.label}</span>
            <IconChevronRight className="w-5 h-5 text-nesio-muted" />
          </button>
        ))}
      </div>
    </div>
  )
}
