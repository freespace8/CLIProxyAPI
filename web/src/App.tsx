import { useCallback, useMemo, useState } from 'react'
import {
  CenteredShell,
  LoginCard,
  PendingCard,
} from '@/components/dashboard/AccessViews'
import {
  DashboardView,
} from '@/components/dashboard/DashboardViews'
import {
  EMPTY_DASHBOARD_DATA,
  STORAGE_KEY,
  readStoredKey,
  summarize,
  useDashboardData,
} from '@/components/dashboard/dashboardState'
import type { Summary } from './types'

export default function App() {
  const [draftKey, setDraftKey] = useState(readStoredKey)
  const [accessKey, setAccessKey] = useState(readStoredKey)
  const { data, error, loading, refresh, updatedAt } = useDashboardData(accessKey)

  const summary = useMemo<Summary>(() => {
    if (!data) return summarize(EMPTY_DASHBOARD_DATA)
    return summarize(data)
  }, [data])

  const handleSubmit = useCallback(() => {
    const trimmedKey = draftKey.trim()
    if (!trimmedKey) return
    window.localStorage.setItem(STORAGE_KEY, trimmedKey)
    setAccessKey(trimmedKey)
  }, [draftKey])

  const handleLogout = useCallback(() => {
    window.localStorage.removeItem(STORAGE_KEY)
    setDraftKey('')
    setAccessKey('')
  }, [])

  if (!accessKey.trim()) {
    return (
      <CenteredShell>
        <LoginCard draftKey={draftKey} error={error} loading={loading} onChange={setDraftKey} onSubmit={handleSubmit} />
      </CenteredShell>
    )
  }

  if (!data) {
    return (
      <CenteredShell>
        <PendingCard error={error} loading={loading} onLogout={handleLogout} onRefresh={refresh} />
      </CenteredShell>
    )
  }

  return (
    <DashboardView
      accessKey={accessKey}
      data={data}
      error={error}
      loading={loading}
      onLogout={handleLogout}
      onRefresh={refresh}
      summary={summary}
      updatedAt={updatedAt}
    />
  )
}
