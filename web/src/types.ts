export interface LiveRequest {
  requestId: string
  model: string
  thinkingLevel?: string
  serviceTier?: string
  startTime: string
}

export interface RequestLogRecord {
  id: number
  timestamp: string
  durationMs: number
  totalTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
  statusCode: number
  success: boolean
  model: string
  thinkingLevel?: string
  serviceTier?: string
  errorMessage?: string
  responseBody?: string
}

export interface RequestLogSnapshotEvent {
  type: 'snapshot'
  requests?: LiveRequest[]
  logs?: RequestLogRecord[]
}

export interface RequestLogAppendEvent {
  type: 'append'
  requestId?: string
  log?: RequestLogRecord
}

export interface LiveRequestUpsertEvent {
  type: 'live_upsert'
  request?: LiveRequest
}

export interface RequestLogHeartbeatEvent {
  type: 'heartbeat'
}

export type RequestLogStreamEvent =
  | RequestLogAppendEvent
  | LiveRequestUpsertEvent
  | RequestLogHeartbeatEvent
  | RequestLogSnapshotEvent
