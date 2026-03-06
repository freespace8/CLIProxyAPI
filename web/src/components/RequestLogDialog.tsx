import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { loadCodexRequestLog } from '../api'
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

function useRequestLogDetail(accessKey: string, logId: number | null) {
  const [detail, setDetail] = useState<RequestLogRecord | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  useEffect(() => {
    if (!logId) return
    let cancelled = false
    setLoading(true)
    setError('')
    void loadCodexRequestLog(accessKey, logId)
      .then((response) => {
        if (!cancelled) setDetail(response.log ?? null)
      })
      .catch((reason) => {
        if (!cancelled) setError(reason instanceof Error ? reason.message : '加载失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [accessKey, logId])

  return { detail, error, loading }
}

function DetailChips(props: { detail: RequestLogRecord }) {
  const statusClassName = props.detail.statusCode === 200 ? 'text-foreground' : 'text-destructive'

  return (
    <div className="flex flex-wrap gap-2">
      <span className="inline-flex items-center rounded-full border px-3 py-1 text-xs">{props.detail.model}</span>
      {props.detail.reasoning ? <span className="inline-flex items-center rounded-full border px-3 py-1 text-xs">{props.detail.reasoning}</span> : null}
      <span className={`inline-flex items-center rounded-full border px-3 py-1 text-xs ${statusClassName}`}>
        {props.detail.statusCode === 200 ? '成功' : `失败(${props.detail.statusCode})`}
      </span>
    </div>
  )
}

export function RequestLogDialog(props: { accessKey: string; logId: number | null; onClose: () => void }) {
  const { detail, error, loading } = useRequestLogDetail(props.accessKey, props.logId)

  useEffect(() => {
    if (!props.logId) return

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') props.onClose()
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [props.logId, props.onClose])

  if (!props.logId) return null

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
                <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">请求详情</p>
                <h2 className="text-lg font-semibold">{detail?.requestId ?? '加载中'}</h2>
              </div>
              {detail ? (
                <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
                  <p className="break-all font-mono">
                    {detail.requestMethod} {detail.requestUrl}
                  </p>
                  <p className="font-mono">{formatTime(detail.timestamp)}</p>
                </div>
              ) : null}
              {detail ? <DetailChips detail={detail} /> : null}
            </div>
            <button className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md border hover:bg-accent" onClick={props.onClose} type="button">
              <X className="size-4" />
            </button>
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-auto px-5 py-4 sm:px-6 sm:py-5">
          {error ? <p className="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p> : null}
          {detail ? (
            <div className="grid gap-4">
              <DetailBlock action="请求头" title="请求头">{JSON.stringify(detail.requestHeaders ?? {}, null, 2)}</DetailBlock>
              <DetailBlock action="请求体" title="请求 Body">{detail.requestBody}</DetailBlock>
              <DetailBlock action="响应体" title="响应 Body">{detail.responseBody}</DetailBlock>
            </div>
          ) : null}
        </div>
      </section>
    </div>
  )
}
