import type {
  DashboardData,
  LiveRequestsResponse,
  RequestLogDetailResponse,
  RequestLogsResponse,
} from './types'

const API_PATHS = {
  config: '/v0/management/config',
  usage: '/v0/management/usage',
  authFiles: '/v0/management/auth-files',
  apiKeys: '/v0/management/api-keys',
  codexLive: '/v0/management/dashboard/codex/live',
  codexLogs: '/v0/management/dashboard/codex/logs',
} as const

async function fetchJSON<T>(path: string, key: string): Promise<T> {
  const response = await fetch(path, {
    headers: { Authorization: `Bearer ${key}` },
  })
  if (!response.ok) {
    const message = (await response.text()).trim()
    throw new Error(message || `请求失败 (${response.status})`)
  }
  return response.json() as Promise<T>
}

export async function loadDashboardData(key: string): Promise<DashboardData> {
  const [config, usage, authFiles, apiKeys] = await Promise.all([
    fetchJSON<DashboardData['config']>(API_PATHS.config, key),
    fetchJSON<DashboardData['usage']>(API_PATHS.usage, key),
    fetchJSON<DashboardData['authFiles']>(API_PATHS.authFiles, key),
    fetchJSON<DashboardData['apiKeys']>(API_PATHS.apiKeys, key),
  ])
  return { config, usage, authFiles, apiKeys }
}

export function loadCodexLiveRequests(key: string): Promise<LiveRequestsResponse> {
  return fetchJSON<LiveRequestsResponse>(API_PATHS.codexLive, key)
}

export function loadCodexRequestLogs(key: string): Promise<RequestLogsResponse> {
  return fetchJSON<RequestLogsResponse>(API_PATHS.codexLogs, key)
}

export function loadCodexRequestLog(key: string, id: number): Promise<RequestLogDetailResponse> {
  return fetchJSON<RequestLogDetailResponse>(`${API_PATHS.codexLogs}/${id}`, key)
}
