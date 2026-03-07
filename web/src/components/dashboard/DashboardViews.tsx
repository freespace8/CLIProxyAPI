import {
  LogOut,
} from 'lucide-react'
import { CodexMonitor } from '@/components/CodexMonitor'
import { Button } from '@/components/ui/button'

function DashboardBar(props: { onLogout: () => void }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="min-w-0">
        <h1 className="truncate text-xl font-semibold sm:text-2xl">CLI Proxy API Dashboard</h1>
      </div>
      <Button onClick={props.onLogout} variant="outline">
        <LogOut className="size-4" />
        退出
      </Button>
    </div>
  )
}

export function DashboardView(props: {
  accessKey: string
  onLogout: () => void
}) {
  return (
    <main className="min-h-screen">
      <div className="mx-auto flex min-h-screen max-w-7xl flex-col gap-6 px-4 py-8 sm:px-6 lg:px-8">
        <DashboardBar onLogout={props.onLogout} />
        <CodexMonitor accessKey={props.accessKey} />
      </div>
    </main>
  )
}
