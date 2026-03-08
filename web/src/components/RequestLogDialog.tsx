import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import type { RequestLogRecord } from '../types'

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

function formatDuration(durationMs?: number): string {
  if (durationMs == null || !Number.isFinite(durationMs)) return '--'
  if (durationMs < 1000) return `${Math.round(durationMs)}ms`
  return `${(durationMs / 1000).toFixed(2)}s`
}

function formatTokensPerSecond(totalTokens: number, durationMs: number): string {
  if (!Number.isFinite(totalTokens) || totalTokens <= 0) return '--'
  if (!Number.isFinite(durationMs) || durationMs <= 0) return '--'
  const tokensPerSecond = totalTokens / (durationMs / 1000)
  if (!Number.isFinite(tokensPerSecond) || tokensPerSecond <= 0) return '--'
  if (tokensPerSecond >= 1000) return `${(tokensPerSecond / 1000).toFixed(1)}K tok/s`
  return `${Math.round(tokensPerSecond)} tok/s`
}

function formatPerformance(detail: RequestLogRecord): string {
  return [
    formatDuration(detail.firstTokenMs),
    formatDuration(detail.durationMs),
    formatTokensPerSecond(detail.totalTokens, detail.durationMs),
  ].join(' / ')
}

function copyText(value: string) {
  if (!value) return
  void navigator.clipboard.writeText(value)
}

function DetailBlock(props: { action?: string; children: string; title: string }) {
  return (
    <section className="grid gap-3">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h3 className="text-sm font-semibold">{props.title}</h3>
        {props.action ? (
          <button
            className="w-fit text-sm text-muted-foreground underline underline-offset-4"
            onClick={() => copyText(props.children)}
            type="button"
          >
            复制{props.action}
          </button>
        ) : null}
      </div>
      <pre className="overflow-auto whitespace-pre-wrap break-words rounded-xl border bg-muted/40 p-4 text-xs text-foreground">{props.children || '无'}</pre>
    </section>
  )
}

function DetailChips(props: { detail: RequestLogRecord }) {
  const statusClassName = props.detail.statusCode === 200 ? 'text-foreground' : 'text-destructive'

  return (
    <div className="flex flex-wrap gap-2">
      <span className="inline-flex items-center rounded-full border px-3 py-1 text-xs">{props.detail.model}</span>
      <span className={`inline-flex items-center rounded-full border px-3 py-1 text-xs ${statusClassName}`}>
        {props.detail.statusCode === 200 ? '成功' : `失败(${props.detail.statusCode})`}
      </span>
    </div>
  )
}

function buildTitle(detail: RequestLogRecord): string {
  return detail.success ? '成功' : `失败(${detail.statusCode})`
}

export function RequestLogDialog(props: { log: RequestLogRecord | null; onClose: () => void }) {
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!props.log) return

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') props.onClose()
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [props.log, props.onClose])

  useEffect(() => {
    if (!copied) return
    const timer = window.setTimeout(() => setCopied(false), 1500)
    return () => window.clearTimeout(timer)
  }, [copied])

  if (!props.log) return null

  const responseBody = props.log.responseBody?.trim() || '无响应内容'

  return (
    <div className="fixed inset-0 z-50 bg-black/40 p-3 sm:p-6 lg:p-8" onClick={props.onClose}>
      <section
        className="mx-auto flex max-h-[92vh] w-full max-w-6xl flex-col overflow-hidden rounded-2xl border bg-background shadow-xl sm:max-h-[88vh]"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="sticky top-0 z-10 border-b bg-background px-4 py-4 sm:px-6 sm:py-4">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div className="min-w-0 space-y-2">
              <div className="flex flex-col gap-1 sm:flex-row sm:flex-wrap sm:items-center sm:gap-x-3 sm:gap-y-1">
                <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">错误响应</p>
                <h2 className="text-base font-semibold sm:text-lg">{buildTitle(props.log)}</h2>
              </div>
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <p className="font-mono">{formatTime(props.log.timestamp)}</p>
                <p className="font-mono">{`性能 ${formatPerformance(props.log)}`}</p>
                <p className="font-mono">Tokens {props.log.totalTokens || 0}</p>
              </div>
              <DetailChips detail={props.log} />
            </div>
            <button className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md border hover:bg-accent" onClick={props.onClose} type="button">
              <X className="size-4" />
            </button>
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-auto px-4 py-4 sm:px-6 sm:py-5">
          <div className="grid gap-4">
            <DetailBlock title="响应内容">{responseBody}</DetailBlock>
            <button
              className="w-fit text-sm text-muted-foreground underline underline-offset-4"
              onClick={() => {
                copyText(responseBody)
                setCopied(true)
              }}
              type="button"
            >
              {copied ? '已复制' : '复制响应内容'}
            </button>
          </div>
        </div>
      </section>
    </div>
  )
}
