import type { FormEvent, ReactNode } from 'react'
import { LogOut, RefreshCw, TriangleAlert } from 'lucide-react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'

export function CenteredShell(props: { children: ReactNode }) {
  return (
    <main className="min-h-screen">
      <div className="mx-auto flex min-h-screen max-w-7xl items-center justify-center px-4 py-10 sm:px-6 lg:px-8">
        {props.children}
      </div>
    </main>
  )
}

export function LoginCard(props: {
  draftKey: string
  error: string
  loading: boolean
  onChange: (value: string) => void
  onSubmit: () => void
}) {
  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    props.onSubmit()
  }

  return (
    <Card className="w-full max-w-2xl border-border/80 bg-card/85">
      <CardHeader className="space-y-4">
        <Badge variant="outline" className="w-fit">Dashboard Access</Badge>
        <div className="space-y-3">
          <CardTitle className="text-4xl sm:text-5xl">CLI Proxy API Dashboard</CardTitle>
          <CardDescription className="max-w-xl text-base">
            输入 Management Key，进入简洁的黑白风格管理页面。
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent className="space-y-5">
        {props.error ? (
          <Alert variant="destructive">
            <TriangleAlert />
            <AlertTitle>连接失败</AlertTitle>
            <AlertDescription>{props.error}</AlertDescription>
          </Alert>
        ) : null}
        <form className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto]" onSubmit={handleSubmit}>
          <Input
            className="h-11"
            type="password"
            value={props.draftKey}
            placeholder="请输入 Management Key"
            onChange={(event) => props.onChange(event.target.value)}
          />
          <Button className="h-11 px-6" disabled={props.loading || !props.draftKey.trim()} type="submit">
            {props.loading ? <RefreshCw className="size-4 animate-spin" /> : null}
            {props.loading ? '连接中...' : '进入 Dashboard'}
          </Button>
        </form>
        <div className="flex flex-wrap gap-2 text-sm text-muted-foreground">
          <Badge variant="outline">Management API</Badge>
          <Badge variant="outline">Codex Live Monitor</Badge>
          <Badge variant="outline">Recent 20 Logs</Badge>
        </div>
      </CardContent>
    </Card>
  )
}

export function PendingCard(props: {
  error: string
  loading: boolean
  onLogout: () => void
  onRefresh: () => void
}) {
  return (
    <Card className="w-full max-w-2xl border-border/80 bg-card/85">
      <CardHeader className="space-y-4">
        <Badge variant="outline" className="w-fit">Dashboard Access</Badge>
        <div className="space-y-2">
          <CardTitle className="text-4xl sm:text-5xl">CLI Proxy API Dashboard</CardTitle>
          <CardDescription className="text-base">
            {props.loading ? '正在读取管理接口数据。' : '尚未获取到 dashboard 数据。'}
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent className="space-y-5">
        {props.error ? (
          <Alert variant="destructive">
            <TriangleAlert />
            <AlertTitle>读取失败</AlertTitle>
            <AlertDescription>{props.error}</AlertDescription>
          </Alert>
        ) : null}
        {props.loading ? (
          <div className="space-y-3">
            <Skeleton className="h-6 w-1/2" />
            <Skeleton className="h-11 w-full" />
            <Skeleton className="h-11 w-2/3" />
          </div>
        ) : null}
        <div className="flex flex-wrap gap-3">
          <Button disabled={props.loading} onClick={props.onRefresh} variant="outline">
            <RefreshCw className="size-4" />
            重试
          </Button>
          <Button onClick={props.onLogout} variant="secondary">
            <LogOut className="size-4" />
            退出
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
