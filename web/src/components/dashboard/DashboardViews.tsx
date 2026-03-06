import {
  Activity,
  Coins,
  LogOut,
  TriangleAlert,
  KeyRound,
  ShieldCheck,
} from 'lucide-react'
import { CodexMonitor } from '@/components/CodexMonitor'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  formatCompactCount,
  formatCount,
  formatSuccessRate,
} from '@/components/dashboard/dashboardState'
import type { DashboardData, Summary } from '@/types'

interface SummaryCardData {
  hint: string
  icon: typeof Activity
  title: string
  value: string
}

function buildSummaryCards(summary: Summary): SummaryCardData[] {
  return [
    {
      title: 'API Keys',
      value: formatCount(summary.apiKeyCount),
      hint: '已配置的顶层 API Key 数量',
      icon: KeyRound,
    },
    {
      title: '认证文件',
      value: formatCount(summary.authCount),
      hint: `启用 ${formatCount(summary.activeAuthCount)} 个`,
      icon: ShieldCheck,
    },
    {
      title: '总请求数',
      value: formatCount(summary.totalRequests),
      hint: `成功率 ${formatSuccessRate(summary)}`,
      icon: Activity,
    },
    {
      title: '总 Tokens',
      value: formatCompactCount(summary.totalTokens),
      hint: `失败 ${formatCount(summary.failureCount)} 次`,
      icon: Coins,
    },
  ]
}

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

function SummaryGrid(props: { summary: Summary }) {
  const cards = buildSummaryCards(props.summary)
  return (
    <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      {cards.map((card) => {
        const Icon = card.icon
        return (
          <Card key={card.title}>
            <CardHeader className="space-y-2 p-4">
              <div className="flex items-center justify-between gap-3">
                <CardDescription className="text-xs">{card.title}</CardDescription>
                <Icon className="size-4 text-muted-foreground" />
              </div>
              <CardTitle className="text-3xl">{card.value}</CardTitle>
              <CardDescription className="text-xs">{card.hint}</CardDescription>
            </CardHeader>
          </Card>
        )
      })}
    </section>
  )
}

export function DashboardView(props: {
  accessKey: string
  data: DashboardData
  error: string
  loading: boolean
  onLogout: () => void
  onRefresh: () => void
  summary: Summary
  updatedAt: string
}) {
  return (
    <main className="min-h-screen">
      <div className="mx-auto flex min-h-screen max-w-7xl flex-col gap-6 px-4 py-8 sm:px-6 lg:px-8">
        {props.error ? (
          <Alert variant="destructive">
            <TriangleAlert />
            <AlertTitle>刷新异常</AlertTitle>
            <AlertDescription>{props.error}</AlertDescription>
          </Alert>
        ) : null}
        <DashboardBar onLogout={props.onLogout} />
        <SummaryGrid summary={props.summary} />
        <CodexMonitor accessKey={props.accessKey} />
      </div>
    </main>
  )
}
