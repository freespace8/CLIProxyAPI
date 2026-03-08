import { formatCompactCount } from './dashboard/dashboardState'
import type { RequestLogRecord } from '../types'

export function formatRequestLogDuration(durationMs?: number): string {
  if (durationMs == null || !Number.isFinite(durationMs)) return '--'
  if (durationMs < 1000) return `${Math.round(durationMs)}ms`
  return `${(durationMs / 1000).toFixed(2)}s`
}

export function formatRequestLogTokensPerSecond(outputTokens: number | undefined, durationMs: number | undefined, firstTokenMs: number | undefined): string {
  if (!Number.isFinite(outputTokens) || outputTokens <= 0) return '--'
  if (!Number.isFinite(durationMs) || durationMs <= 0) return '--'
  if (!Number.isFinite(firstTokenMs) || firstTokenMs == null || firstTokenMs < 0) return '--'

  const generationDurationMs = durationMs - firstTokenMs
  if (!Number.isFinite(generationDurationMs) || generationDurationMs <= 0) return '--'

  const tokensPerSecond = outputTokens / (generationDurationMs / 1000)
  if (!Number.isFinite(tokensPerSecond) || tokensPerSecond <= 0) return '--'

  return `${formatCompactCount(Math.round(tokensPerSecond))}tok/s`
}

export function formatRequestLogPerformance(log: Pick<RequestLogRecord, 'success' | 'firstTokenMs' | 'durationMs' | 'outputTokens'>): string {
  if (!log.success) return '--'
  return [
    formatRequestLogDuration(log.firstTokenMs),
    formatRequestLogDuration(log.durationMs),
    formatRequestLogTokensPerSecond(log.outputTokens, log.durationMs, log.firstTokenMs),
  ].join('/')
}
