import {
  LogOut,
} from 'lucide-react'
import { CodexMonitor } from '@/components/CodexMonitor'
import { Button } from '@/components/ui/button'

function DashboardBar(props: { onLogout: () => void }) {
  return (
    <div className="flex flex-col gap-4 rounded-2xl border bg-card/70 p-4 shadow-sm sm:flex-row sm:items-center sm:justify-between sm:p-5">
      <div className="min-w-0 space-y-1">
        <h1 className="truncate text-xl font-semibold sm:text-2xl lg:text-3xl">CLI Proxy API Dashboard</h1>
      </div>
      <Button className="w-full sm:w-auto" onClick={props.onLogout} variant="outline">
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
    <main className="min-h-screen bg-background">
      <div className="mx-auto flex min-h-screen max-w-7xl flex-col gap-5 px-4 py-5 sm:gap-6 sm:px-6 sm:py-8 lg:px-8 lg:py-10">
        <DashboardBar onLogout={props.onLogout} />
        <CodexMonitor accessKey={props.accessKey} />
      </div>
    </main>
  )
}
