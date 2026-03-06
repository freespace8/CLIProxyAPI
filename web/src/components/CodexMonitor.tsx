import { useCallback, useEffect, useState } from 'react'
import { loadCodexLiveRequests, loadCodexRequestLogs } from '../api'
import { formatCompactCount } from './dashboard/dashboardState'
import type { LiveRequest, RequestLogRecord } from '../types'
import { RequestLogDialog } from './RequestLogDialog'

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

function formatElapsed(startTime: string, now: number): string {
  const startedAt = new Date(startTime).getTime()
  if (!Number.isFinite(startedAt)) return '--'
  return `${((now - startedAt) / 1000).toFixed(1)}s`
}

function formatDuration(durationMs: number): string {
  if (!Number.isFinite(durationMs)) return '--'
  if (durationMs < 1000) return `${Math.round(durationMs)}ms`
  return `${(durationMs / 1000).toFixed(2)}s`
}

function formatCacheTokens(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '--'
  return formatCompactCount(value)
}

function formatTokenCount(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '--'
  return formatCompactCount(value)
}

function statusClass(statusCode: number): string {
  if (statusCode >= 400) return 'text-destructive'
  return 'text-foreground'
}

function requestLabel(request: Pick<LiveRequest, 'model' | 'reasoning'> | Pick<RequestLogRecord, 'model' | 'reasoning'>): string {
  return request.reasoning ? `${request.model} · ${request.reasoning}` : request.model
}

function SectionHeader(props: {
  description: string
  title: string
}) {
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="space-y-1">
        <h2 className="text-xl font-semibold tracking-tight">{props.title}</h2>
        <p className="text-sm text-muted-foreground">{props.description}</p>
      </div>
    </div>
  )
}

function LiveRequestItem(props: { now: number; request: LiveRequest }) {
  const { request } = props
  return (
    <article className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-2 rounded-lg border p-3">
      <div className="min-w-0 space-y-1.5">
        <div className="flex flex-wrap items-center gap-2">
          <strong className="text-sm font-semibold">{requestLabel(request)}</strong>
        </div>
        <p className="break-all font-mono text-xs text-muted-foreground">
          {request.requestMethod} {request.requestUrl}
        </p>
        <p className="break-all font-mono text-xs text-muted-foreground">{request.requestId}</p>
      </div>
      <div className="shrink-0 text-right font-mono text-xs text-muted-foreground">
        <div>{formatElapsed(request.startTime, props.now)}</div>
      </div>
    </article>
  )
}

function LogStatusText(props: { statusCode: number }) {
  const label = props.statusCode === 200 ? '成功' : `失败(${props.statusCode})`
  return <span className={`text-sm font-medium ${statusClass(props.statusCode)}`}>{label}</span>
}

function LogsTable(props: {
  logs: RequestLogRecord[]
  onSelect: (id: number) => void
}) {
  return (
    <div className="mt-4 overflow-x-auto">
      <div className="min-w-[1020px]">
        <div className="grid grid-cols-[132px_minmax(240px,1fr)_88px_100px_88px_100px_100px_76px] items-center gap-4 border-b px-2 py-3 text-[11px] uppercase tracking-[0.18em] text-muted-foreground">
          <span>时间</span>
          <span>模型</span>
          <span>状态</span>
          <span>耗时</span>
          <span>Token</span>
          <span>读缓存</span>
          <span>写缓存</span>
          <span>详情</span>
        </div>
        {props.logs.length === 0 ? <p className="px-2 py-8 text-sm text-muted-foreground">最近还没有 Codex 请求日志。</p> : null}
        {props.logs.map((log) => (
          <div className="grid grid-cols-[132px_minmax(240px,1fr)_88px_100px_88px_100px_100px_76px] items-center gap-4 border-b px-2 py-4 last:border-b-0" key={log.id}>
            <span className="font-mono text-xs text-muted-foreground">{formatTime(log.timestamp)}</span>
            <span className="min-w-0">
              <span className="block truncate text-sm font-semibold" title={log.requestUrl}>
                {requestLabel(log)}
              </span>
              <span className="block truncate font-mono text-xs text-muted-foreground">{log.requestId}</span>
            </span>
            <span><LogStatusText statusCode={log.statusCode} /></span>
            <span className="font-mono text-xs">{formatDuration(log.durationMs)}</span>
            <span className="font-mono text-xs">{formatTokenCount(log.totalTokens)}</span>
            <span className="font-mono text-xs">{formatCacheTokens(log.cacheReadTokens)}</span>
            <span className="font-mono text-xs">{formatCacheTokens(log.cacheWriteTokens)}</span>
            <span>
              <button
                className="text-sm font-medium text-foreground underline underline-offset-4 focus-visible:outline-none"
                onClick={() => props.onSelect(log.id)}
                type="button"
              >
                查看
              </button>
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

function useCodexMonitorState(accessKey: string) {
  const [liveRequests, setLiveRequests] = useState<LiveRequest[]>([])
  const [requestLogs, setRequestLogs] = useState<RequestLogRecord[]>([])
  const [liveError, setLiveError] = useState('')
  const [logError, setLogError] = useState('')
  const refreshLive = useCallback(async () => {
    setLiveError('')
    try {
      const response = await loadCodexLiveRequests(accessKey)
      setLiveRequests(response.requests ?? [])
    } catch (error) {
      setLiveError(error instanceof Error ? error.message : '加载失败')
      setLiveRequests([])
    }
  }, [accessKey])

  const refreshLogs = useCallback(async () => {
    setLogError('')
    try {
      const response = await loadCodexRequestLogs(accessKey)
      setRequestLogs(response.logs ?? [])
    } catch (error) {
      setLogError(error instanceof Error ? error.message : '加载失败')
      setRequestLogs([])
    }
  }, [accessKey])

  useEffect(() => {
    void refreshLive()
    void refreshLogs()
  }, [refreshLive, refreshLogs])

  useEffect(() => {
    const liveTimer = window.setInterval(() => {
      void refreshLive()
    }, 2000)
    const logTimer = window.setInterval(() => {
      void refreshLogs()
    }, 5000)
    return () => {
      window.clearInterval(liveTimer)
      window.clearInterval(logTimer)
    }
  }, [refreshLive, refreshLogs])

  return {
    liveError,
    liveRequests,
    logError,
    refreshLive,
    refreshLogs,
    requestLogs,
  }
}

export function CodexMonitor(props: { accessKey: string }) {
  const [selectedLogId, setSelectedLogId] = useState<number | null>(null)
  const [now, setNow] = useState(() => Date.now())
  const {
    liveError,
    liveRequests,
    logError,
    requestLogs,
  } = useCodexMonitorState(props.accessKey)

  useEffect(() => {
    const tickTimer = window.setInterval(() => setNow(Date.now()), 200)
    return () => window.clearInterval(tickTimer)
  }, [])

  return (
    <section className="grid gap-6">
      <section className="rounded-2xl border bg-card p-5 shadow-sm sm:p-6">
        <SectionHeader
          description="展示当前正在执行的 Responses 请求，并以秒级更新耗时。"
          title="请求监控"
        />
        {liveError ? <p className="mt-4 rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">{liveError}</p> : null}
        <div className="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
          {liveRequests.length === 0 ? <p className="rounded-xl border border-dashed px-4 py-8 text-sm text-muted-foreground">当前无进行中请求</p> : null}
          {liveRequests.map((request) => (
            <LiveRequestItem key={request.requestId} now={now} request={request} />
          ))}
        </div>
      </section>

      <section className="rounded-2xl border bg-card p-5 shadow-sm sm:p-6">
        <SectionHeader
          description="保留最近 20 条请求，支持查看原始请求与上游返回详情。"
          title="请求日志"
        />
        {logError ? <p className="mt-4 rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">{logError}</p> : null}
        <LogsTable logs={requestLogs} onSelect={setSelectedLogId} />
      </section>

      <RequestLogDialog accessKey={props.accessKey} logId={selectedLogId} onClose={() => setSelectedLogId(null)} />
    </section>
  )
}
