import { useEffect, useMemo, useState } from 'react'
import { openCodexRequestLogStream, StreamRequestError } from '../api'
import { formatCompactCount } from './dashboard/dashboardState'
import type { LiveRequest, RequestLogRecord, RequestLogStreamEvent } from '../types'
import { RequestLogDialog } from './RequestLogDialog'
import { Badge } from './ui/badge'

const MAX_VISIBLE_LOGS = 20
const INITIAL_RECONNECT_DELAY_MS = 1500
const MAX_RECONNECT_DELAY_MS = 10000

type StreamState = 'connecting' | 'connected' | 'reconnecting' | 'stopped'

function formatTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(date)
}

function formatDuration(durationMs: number): string {
  if (!Number.isFinite(durationMs)) return '--'
  if (durationMs < 1000) return `${Math.round(durationMs)}ms`
  return `${(durationMs / 1000).toFixed(2)}s`
}

function formatElapsed(startTime: string, now: number): string {
  const startedAt = new Date(startTime).getTime()
  if (!Number.isFinite(startedAt)) return '--'
  const elapsedMs = Math.max(0, now - startedAt)
  const steppedTenths = Math.floor(elapsedMs / 100)
  const totalSeconds = Math.floor(steppedTenths / 10)
  const tenths = steppedTenths % 10
  const seconds = totalSeconds % 60
  const minutes = Math.floor(totalSeconds / 60)

  if (minutes < 100) {
    return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}.${tenths}`
  }

  return `${minutes}:${String(seconds).padStart(2, '0')}.${tenths}`
}

function formatModelWithThinking(model: string, thinkingLevel?: string): string {
  const normalizedModel = model.trim() || '--'
  const normalizedThinkingLevel = thinkingLevel?.trim()
  if (!normalizedThinkingLevel) return normalizedModel
  return `${normalizedModel} ${normalizedThinkingLevel}`
}

function formatTokenCount(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '--'
  return formatCompactCount(value)
}

function formatStatusLabel(log: RequestLogRecord): string {
  if (log.success) return '成功'
  return `失败(${log.statusCode})`
}

function statusClass(log: RequestLogRecord): string {
  return log.success ? 'text-foreground' : 'text-destructive'
}

function sortLogs(logs: RequestLogRecord[]): RequestLogRecord[] {
  return [...logs].sort((left, right) => {
    const leftTime = new Date(left.timestamp).getTime()
    const rightTime = new Date(right.timestamp).getTime()
    if (Number.isFinite(leftTime) && Number.isFinite(rightTime) && leftTime !== rightTime) {
      return rightTime - leftTime
    }
    return right.id - left.id
  })
}

function mergeLogs(currentLogs: RequestLogRecord[], nextLog: RequestLogRecord): RequestLogRecord[] {
  const merged = [nextLog, ...currentLogs.filter((log) => log.id !== nextLog.id)]
  return sortLogs(merged).slice(0, MAX_VISIBLE_LOGS)
}

function normalizeSnapshot(logs: RequestLogRecord[]): RequestLogRecord[] {
  return sortLogs(logs).slice(0, MAX_VISIBLE_LOGS)
}

function sortLiveRequests(requests: LiveRequest[]): LiveRequest[] {
  return [...requests].sort((left, right) => {
    const leftTime = new Date(left.startTime).getTime()
    const rightTime = new Date(right.startTime).getTime()
    if (Number.isFinite(leftTime) && Number.isFinite(rightTime) && leftTime !== rightTime) {
      return rightTime - leftTime
    }
    return left.requestId.localeCompare(right.requestId)
  })
}

function mergeLiveRequests(currentRequests: LiveRequest[], nextRequest: LiveRequest): LiveRequest[] {
  return sortLiveRequests([nextRequest, ...currentRequests.filter((request) => request.requestId !== nextRequest.requestId)])
}

function removeLiveRequest(currentRequests: LiveRequest[], requestId: string): LiveRequest[] {
  return currentRequests.filter((request) => request.requestId !== requestId)
}

async function readNdjsonStream(
  stream: ReadableStream<Uint8Array>,
  onEvent: (event: RequestLogStreamEvent) => void,
) {
  const reader = stream.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() ?? ''

      for (const rawLine of lines) {
        const line = rawLine.trim()
        if (!line) continue
        onEvent(JSON.parse(line) as RequestLogStreamEvent)
      }
    }

    const tail = `${buffer}${decoder.decode()}`.trim()
    if (tail) onEvent(JSON.parse(tail) as RequestLogStreamEvent)
  } finally {
    reader.releaseLock()
  }
}

function streamStateLabel(state: StreamState): string {
  if (state === 'connected') return '已连接'
  if (state === 'reconnecting') return '重连中'
  if (state === 'stopped') return '已停止'
  return '连接中'
}

function useCodexRequestLogStream(accessKey: string) {
  const [liveRequests, setLiveRequests] = useState<LiveRequest[]>([])
  const [requestLogs, setRequestLogs] = useState<RequestLogRecord[]>([])
  const [streamError, setStreamError] = useState('')
  const [streamState, setStreamState] = useState<StreamState>('connecting')
  const [lastEventAt, setLastEventAt] = useState('')

  useEffect(() => {
    if (!accessKey.trim()) {
      setLiveRequests([])
      setRequestLogs([])
      setStreamError('')
      setStreamState('stopped')
      setLastEventAt('')
      return
    }

    const abortController = new AbortController()
    let reconnectTimer: number | null = null
    let reconnectDelay = INITIAL_RECONNECT_DELAY_MS
    let isClosed = false

    const clearReconnectTimer = () => {
      if (reconnectTimer) {
        window.clearTimeout(reconnectTimer)
        reconnectTimer = null
      }
    }

    const scheduleReconnect = (message: string) => {
      if (isClosed) return
      clearReconnectTimer()
      setStreamError(message)
      setStreamState('reconnecting')
      reconnectTimer = window.setTimeout(() => {
        reconnectDelay = Math.min(reconnectDelay * 2, MAX_RECONNECT_DELAY_MS)
        void connect()
      }, reconnectDelay)
    }

    const handleStreamEvent = (event: RequestLogStreamEvent) => {
      const eventTime = new Date().toISOString()
      setLastEventAt(eventTime)
      setStreamError('')
      setStreamState('connected')

      if (event.type === 'snapshot') {
        setLiveRequests(sortLiveRequests(event.requests ?? []))
        setRequestLogs(normalizeSnapshot(event.logs ?? []))
        return
      }

      if (event.type === 'live_upsert' && event.request) {
        setLiveRequests((currentRequests) => mergeLiveRequests(currentRequests, event.request!))
        return
      }

      if (event.type === 'append' && event.log) {
        if (event.requestId) {
          setLiveRequests((currentRequests) => removeLiveRequest(currentRequests, event.requestId!))
        }
        setRequestLogs((currentLogs) => mergeLogs(currentLogs, event.log!))
      }
    }

    const connect = async () => {
      if (isClosed) return
      setStreamState((currentState) => (currentState === 'connected' ? 'reconnecting' : 'connecting'))

      try {
        const stream = await openCodexRequestLogStream(accessKey, abortController.signal)
        if (isClosed) return
        reconnectDelay = INITIAL_RECONNECT_DELAY_MS
        setStreamError('')
        setStreamState('connected')
        await readNdjsonStream(stream, handleStreamEvent)

        if (!abortController.signal.aborted) {
          scheduleReconnect('日志推送已断开，正在重连…')
        }
      } catch (error) {
        if (abortController.signal.aborted || isClosed) return

        const message = error instanceof Error ? error.message : '日志订阅失败'
        if (error instanceof StreamRequestError && !error.retryable) {
          setStreamError(message)
          setStreamState('stopped')
          return
        }
        scheduleReconnect(message || '日志订阅失败，正在重连…')
      }
    }

    void connect()

    return () => {
      isClosed = true
      clearReconnectTimer()
      abortController.abort()
    }
  }, [accessKey])

  return {
    liveRequests,
    lastEventAt,
    requestLogs,
    streamError,
    streamState,
  }
}

function MetricPair(props: { label: string; value: string; valueClassName?: string }) {
  return (
    <div className="grid gap-1 rounded-lg bg-muted/40 px-3 py-2">
      <span className="text-[11px] font-medium uppercase tracking-[0.14em] text-muted-foreground">{props.label}</span>
      <span className={`min-w-0 text-sm font-medium ${props.valueClassName ?? ''}`.trim()}>{props.value}</span>
    </div>
  )
}

function DashboardHeader(props: {
  lastEventAt: string
  liveCount: number
  state: StreamState
}) {
  const lastEventLabel = props.lastEventAt ? `最近事件 ${formatTime(props.lastEventAt)}` : '等待首包快照'

  return (
    <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold tracking-tight sm:text-xl">请求监控</h2>
      </div>
      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center lg:max-w-[28rem] lg:justify-end">
        <div className="flex flex-wrap gap-2">
          <Badge className="h-8 px-3" variant="outline">{`进行中 ${props.liveCount}`}</Badge>
          <Badge className="h-8 px-3" variant="outline">{streamStateLabel(props.state)}</Badge>
        </div>
        <span className="inline-flex min-h-8 items-center rounded-full border px-3 py-1 text-left font-mono text-[11px] tabular-nums text-muted-foreground sm:text-xs lg:justify-end">
          {lastEventLabel}
        </span>
      </div>
    </div>
  )
}

function LiveRequestItem(props: { now: number; request: LiveRequest }) {
  const modelLabel = formatModelWithThinking(props.request.model, props.request.thinkingLevel)

  return (
    <article className="grid min-h-[60px] w-full grid-cols-[minmax(0,1fr)_78px] items-center gap-2 rounded-lg border px-3 py-2.5 sm:w-[240px]">
      <div className="min-w-0">
        <p className="truncate text-[13px] font-semibold leading-5" title={modelLabel}>
          {modelLabel}
        </p>
        <p className="truncate font-mono tabular-nums text-[11px] text-muted-foreground">{formatTime(props.request.startTime)}</p>
      </div>
      <div className="flex justify-end">
        <span className="inline-flex min-w-[7ch] justify-end whitespace-nowrap font-mono tabular-nums text-[11px] leading-none text-muted-foreground">
          {formatElapsed(props.request.startTime, props.now)}
        </span>
      </div>
    </article>
  )
}

function LiveRequestsPanel(props: { now: number; requests: LiveRequest[] }) {
  return (
    <section className="rounded-2xl border bg-card p-4 shadow-sm sm:p-6">
      <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
        <h3 className="text-base font-semibold tracking-tight">实时请求</h3>
      </div>
      <div className="mt-4 flex flex-wrap gap-3" data-testid="live-requests-grid">
        {props.requests.length === 0 ? <p className="w-full rounded-xl border border-dashed px-4 py-8 text-sm text-muted-foreground">当前无进行中请求</p> : null}
        {props.requests.map((request) => (
          <LiveRequestItem key={request.requestId} now={props.now} request={request} />
        ))}
      </div>
    </section>
  )
}

function MobileLogCard(props: {
  log: RequestLogRecord
  onOpenError: (log: RequestLogRecord) => void
}) {
  const modelLabel = formatModelWithThinking(props.log.model, props.log.thinkingLevel)
  const statusLabel = formatStatusLabel(props.log)

  return (
    <article className="rounded-xl border p-4 shadow-sm">
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0 space-y-1">
            <p className="truncate text-sm font-semibold" title={modelLabel}>{modelLabel}</p>
            <p className="font-mono text-xs text-muted-foreground">{formatTime(props.log.timestamp)}</p>
          </div>
          {props.log.success ? (
            <span className={`inline-flex w-fit items-center rounded-full border px-3 py-1 text-xs font-medium ${statusClass(props.log)}`}>
              {statusLabel}
            </span>
          ) : (
            <button
              className={`inline-flex w-fit items-center rounded-full border px-3 py-1 text-left text-xs font-medium underline underline-offset-4 ${statusClass(props.log)}`}
              onClick={() => props.onOpenError(props.log)}
              title={statusLabel}
              type="button"
            >
              {statusLabel}
            </button>
          )}
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <MetricPair label="耗时" value={formatDuration(props.log.durationMs)} valueClassName="font-mono text-xs tabular-nums sm:text-sm" />
          <MetricPair label="总 Token" value={formatTokenCount(props.log.totalTokens)} valueClassName="font-mono text-xs tabular-nums sm:text-sm" />
          <MetricPair label="缓存读取" value={formatTokenCount(props.log.cacheReadTokens)} valueClassName="font-mono text-xs tabular-nums sm:text-sm" />
          <MetricPair label="缓存写入" value={formatTokenCount(props.log.cacheWriteTokens)} valueClassName="font-mono text-xs tabular-nums sm:text-sm" />
        </div>
      </div>
    </article>
  )
}

function LogsTable(props: {
  logs: RequestLogRecord[]
  onOpenError: (log: RequestLogRecord) => void
}) {
  const desktopGridClassName = 'grid w-full grid-cols-[minmax(0,1.2fr)_minmax(0,1.5fr)_minmax(0,1.4fr)_minmax(72px,0.7fr)_minmax(64px,0.65fr)_minmax(64px,0.7fr)_minmax(64px,0.7fr)] items-center gap-3 lg:gap-4'

  return (
    <>
      <div className="mt-4 grid gap-3 md:hidden">
        {props.logs.length === 0 ? <p className="rounded-xl border border-dashed px-4 py-8 text-sm text-muted-foreground">最近还没有 Codex 请求日志。</p> : null}
        {props.logs.map((log) => (
          <MobileLogCard key={log.id} log={log} onOpenError={props.onOpenError} />
        ))}
      </div>

      <div className="mt-4 hidden md:block" data-testid="logs-desktop-table">
        <div className={`${desktopGridClassName} border-b px-2 py-3 text-[11px] uppercase tracking-[0.18em] text-muted-foreground`} data-testid="logs-desktop-grid">
          <span>时间</span>
          <span>模型</span>
          <span>状态</span>
          <span>耗时</span>
          <span>Token</span>
          <span>读缓存</span>
          <span>写缓存</span>
        </div>
        {props.logs.length === 0 ? <p className="px-2 py-8 text-sm text-muted-foreground">最近还没有 Codex 请求日志。</p> : null}
        {props.logs.map((log) => {
          const modelLabel = formatModelWithThinking(log.model, log.thinkingLevel)

          return (
            <div className={`${desktopGridClassName} border-b px-2 py-4 last:border-b-0`} key={log.id}>
              <span className="truncate font-mono text-xs text-muted-foreground">{formatTime(log.timestamp)}</span>
              <span className="truncate text-sm font-semibold" title={modelLabel}>
                {modelLabel}
              </span>
              <span className="min-w-0">
                {log.success ? (
                  <span className={`block truncate text-sm font-medium ${statusClass(log)}`}>{formatStatusLabel(log)}</span>
                ) : (
                  <button
                    className={`block w-full truncate text-left text-sm font-medium underline underline-offset-4 focus-visible:outline-none ${statusClass(log)}`}
                    onClick={() => props.onOpenError(log)}
                    title={formatStatusLabel(log)}
                    type="button"
                  >
                    {formatStatusLabel(log)}
                  </button>
                )}
              </span>
              <span className="truncate font-mono text-xs">{formatDuration(log.durationMs)}</span>
              <span className="truncate font-mono text-xs">{formatTokenCount(log.totalTokens)}</span>
              <span className="truncate font-mono text-xs">{formatTokenCount(log.cacheReadTokens)}</span>
              <span className="truncate font-mono text-xs">{formatTokenCount(log.cacheWriteTokens)}</span>
            </div>
          )
        })}
      </div>
    </>
  )
}

export function CodexMonitor(props: { accessKey: string }) {
  const [selectedLog, setSelectedLog] = useState<RequestLogRecord | null>(null)
  const [now, setNow] = useState(() => Date.now())
  const { liveRequests, lastEventAt, requestLogs, streamError, streamState } = useCodexRequestLogStream(props.accessKey)

  useEffect(() => {
    if (liveRequests.length === 0) return
    const timer = window.setInterval(() => setNow(Date.now()), 100)
    return () => window.clearInterval(timer)
  }, [liveRequests.length])

  const dialogLog = useMemo(() => {
    if (!selectedLog) return null
    return requestLogs.find((log) => log.id === selectedLog.id) ?? selectedLog
  }, [requestLogs, selectedLog])

  return (
    <section className="grid gap-5 lg:gap-6">
      <section className="rounded-2xl border bg-card p-4 shadow-sm sm:p-6">
        <DashboardHeader lastEventAt={lastEventAt} liveCount={liveRequests.length} state={streamState} />
        {streamError ? <p className="mt-4 rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">{streamError}</p> : null}
      </section>

      <LiveRequestsPanel now={now} requests={liveRequests} />

      <section className="rounded-2xl border bg-card p-4 shadow-sm sm:p-6">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
          <h3 className="text-base font-semibold tracking-tight">最近日志</h3>
        </div>
        <LogsTable logs={requestLogs} onOpenError={setSelectedLog} />
      </section>

      <RequestLogDialog log={dialogLog} onClose={() => setSelectedLog(null)} />
    </section>
  )
}
