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

function copyText(value: string) {
  if (!value) return
  void navigator.clipboard.writeText(value)
}

function DetailBlock(props: { action?: string; children: string; title: string }) {
  return (
    <section className="grid gap-3">
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-semibold">{props.title}</h3>
        {props.action ? (
          <button
            className="text-sm text-muted-foreground underline underline-offset-4"
            onClick={() => copyText(props.children)}
            type="button"
          >
            复制{props.action}
          </button>
        ) : null}
      </div>
      <pre className="overflow-auto rounded-xl border bg-muted/40 p-4 text-xs text-foreground">{props.children || '无'}</pre>
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
  return detail.errorMessage?.trim() || `失败(${detail.statusCode})`
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

  const responseBody = props.log.responseBody?.trim() || '无 responseBody'

  return (
    <div className="fixed inset-0 z-50 bg-black/40 p-4 sm:p-8" onClick={props.onClose}>
      <section
        className="mx-auto flex max-h-[88vh] w-full max-w-6xl flex-col overflow-hidden rounded-2xl border bg-background shadow-xl"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="sticky top-0 z-10 border-b bg-background px-5 py-4 sm:px-6 sm:py-4">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0 space-y-2">
              <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">错误响应</p>
                <h2 className="text-lg font-semibold">{buildTitle(props.log)}</h2>
              </div>
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <p className="font-mono">{formatTime(props.log.timestamp)}</p>
                <p className="font-mono">{Math.round(props.log.durationMs)}ms</p>
                <p className="font-mono">Tokens {props.log.totalTokens || 0}</p>
              </div>
              <DetailChips detail={props.log} />
            </div>
            <button className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md border hover:bg-accent" onClick={props.onClose} type="button">
              <X className="size-4" />
            </button>
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-auto px-5 py-4 sm:px-6 sm:py-5">
          <div className="grid gap-4">
            <DetailBlock title="responseBody">{responseBody}</DetailBlock>
            {props.log.errorMessage ? <p className="text-sm text-muted-foreground">错误信息：{props.log.errorMessage}</p> : null}
            <button
              className="w-fit text-sm text-muted-foreground underline underline-offset-4"
              onClick={() => {
                copyText(responseBody)
                setCopied(true)
              }}
              type="button"
            >
              {copied ? '已复制' : '复制 responseBody'}
            </button>
          </div>
        </div>
      </section>
    </div>
  )
}
