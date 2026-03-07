import { useCallback, useState } from 'react'
import {
  CenteredShell,
  LoginCard,
} from '@/components/dashboard/AccessViews'
import {
  DashboardView,
} from '@/components/dashboard/DashboardViews'
import {
  STORAGE_KEY,
  readStoredKey,
} from '@/components/dashboard/dashboardState'

export default function App() {
  const [draftKey, setDraftKey] = useState(readStoredKey)
  const [accessKey, setAccessKey] = useState(readStoredKey)

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
        <LoginCard draftKey={draftKey} error="" loading={false} onChange={setDraftKey} onSubmit={handleSubmit} />
      </CenteredShell>
    )
  }

  return (
    <DashboardView
      accessKey={accessKey}
      onLogout={handleLogout}
    />
  )
}
