export interface ConfigSnapshot {
  debug?: boolean
  'usage-statistics-enabled'?: boolean
  'logging-to-file'?: boolean
  'request-retry'?: number
  'proxy-url'?: string
  routing?: {
    strategy?: string
  }
}

export interface UsageModelStats {
  total_requests?: number
  total_tokens?: number
}

export interface UsageApiStats {
  models?: Record<string, UsageModelStats>
}

export interface UsageSnapshot {
  total_requests?: number
  success_count?: number
  failure_count?: number
  total_tokens?: number
  apis?: Record<string, UsageApiStats>
}

export interface UsageResponse {
  usage?: UsageSnapshot
}

export interface AuthFileEntry {
  name?: string
  disabled?: boolean
}

export interface AuthFilesResponse {
  files?: AuthFileEntry[]
}

export interface ApiKeysResponse {
  'api-keys'?: string[]
}

export interface DashboardData {
  config: ConfigSnapshot
  usage: UsageResponse
  authFiles: AuthFilesResponse
  apiKeys: ApiKeysResponse
}

export interface Summary {
  apiKeyCount: number
  authCount: number
  activeAuthCount: number
  totalRequests: number
  successCount: number
  failureCount: number
  totalTokens: number
}

export interface LiveRequest {
  requestId: string
  requestMethod: string
  requestUrl: string
  model: string
  reasoning: string
  startTime: string
  isStreaming: boolean
}

export interface LiveRequestsResponse {
  requests?: LiveRequest[]
  count?: number
}

export interface RequestLogRecord {
  id: number
  requestId: string
  requestMethod: string
  requestUrl: string
  requestHeaders: Record<string, string>
  requestBody: string
  responseBody: string
  upstreamRequest: string
  upstreamResponse: string
  timestamp: string
  durationMs: number
  totalTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
  statusCode: number
  success: boolean
  model: string
  reasoning: string
  errorMessage?: string
  isStreaming: boolean
}

export interface RequestLogsResponse {
  logs?: RequestLogRecord[]
  total?: number
}

export interface RequestLogDetailResponse {
  log?: RequestLogRecord
}
