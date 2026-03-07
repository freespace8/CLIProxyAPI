const API_PATHS = {
  codexStream: '/v0/management/dashboard/codex/stream',
} as const

export class StreamRequestError extends Error {
  retryable: boolean

  constructor(message: string, options?: { retryable?: boolean }) {
    super(message)
    this.name = 'StreamRequestError'
    this.retryable = options?.retryable ?? true
  }
}

export async function openCodexRequestLogStream(key: string, signal: AbortSignal): Promise<ReadableStream<Uint8Array>> {
  const requestInit: RequestInit & { priority?: 'low' | 'auto' | 'high' } = {
    cache: 'no-store',
    headers: {
      Accept: 'application/x-ndjson',
      Authorization: `Bearer ${key}`,
    },
    signal,
  }
  requestInit.priority = 'low'

  const response = await fetch(API_PATHS.codexStream, requestInit)
  if (!response.ok) {
    const message = (await response.text()).trim()
    throw new StreamRequestError(message || `请求失败 (${response.status})`, {
      retryable: response.status >= 500 || response.status === 408 || response.status === 429,
    })
  }
  if (!response.body) {
    throw new StreamRequestError('当前环境不支持流式读取管理日志', { retryable: false })
  }
  return response.body
}
