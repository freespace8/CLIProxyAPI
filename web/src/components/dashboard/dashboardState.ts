import { useCallback, useEffect, useState } from 'react'
import { loadDashboardData } from '@/api'
import type { DashboardData, Summary } from '@/types'

const PERCENT_BASE = 100

export const STORAGE_KEY = 'cli-proxy-dashboard-key'
export const EMPTY_DASHBOARD_DATA: DashboardData = {
  config: {},
  usage: {},
  authFiles: {},
  apiKeys: {},
}

interface LoadState {
  data: DashboardData | null
  error: string
  loading: boolean
  updatedAt: string
}

function createInitialLoadState(): LoadState {
  return { data: null, error: '', loading: false, updatedAt: '' }
}

export function readStoredKey(): string {
  return window.localStorage.getItem(STORAGE_KEY) ?? ''
}

export function summarize(data: DashboardData): Summary {
  const usage = data.usage.usage ?? {}
  const authFiles = data.authFiles.files ?? []
  const apiKeys = data.apiKeys['api-keys'] ?? []
  return {
    apiKeyCount: apiKeys.length,
    authCount: authFiles.length,
    activeAuthCount: authFiles.filter((entry) => !entry.disabled).length,
    totalRequests: Number(usage.total_requests ?? 0),
    successCount: Number(usage.success_count ?? 0),
    failureCount: Number(usage.failure_count ?? 0),
    totalTokens: Number(usage.total_tokens ?? 0),
  }
}

export function formatCount(value: number): string {
  return value.toLocaleString('zh-CN')
}

export function formatCompactCount(value: number): string {
  const formatter = new Intl.NumberFormat('en', {
    maximumFractionDigits: 1,
    notation: 'compact',
  })
  return formatter.format(value)
}

export function formatSuccessRate(summary: Summary): string {
  if (summary.totalRequests === 0) return '0%'
  const rate = Math.round((summary.successCount / summary.totalRequests) * PERCENT_BASE)
  return `${rate}%`
}

export function useDashboardData(accessKey: string) {
  const [state, setState] = useState<LoadState>(createInitialLoadState)

  const refresh = useCallback(async () => {
    if (!accessKey.trim()) return
    setState((current) => ({ ...current, error: '', loading: true }))
    try {
      const data = await loadDashboardData(accessKey)
      setState({
        data,
        error: '',
        loading: false,
        updatedAt: new Date().toLocaleString('zh-CN'),
      })
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载失败'
      setState((current) => ({ ...current, error: message, loading: false }))
    }
  }, [accessKey])

  useEffect(() => {
    if (accessKey.trim()) return
    setState(createInitialLoadState())
  }, [accessKey])

  useEffect(() => {
    if (!accessKey.trim()) return
    void refresh()
  }, [accessKey, refresh])

  return { ...state, refresh }
}
