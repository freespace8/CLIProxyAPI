import type { FormEvent, ReactNode } from 'react'
import { RefreshCw, TriangleAlert } from 'lucide-react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'

export function CenteredShell(props: { children: ReactNode }) {
  return (
    <main className="min-h-screen bg-background">
      <div className="mx-auto flex min-h-screen max-w-7xl items-center justify-center px-4 py-6 sm:px-6 sm:py-10 lg:px-8 lg:py-12">
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
    <Card className="w-full max-w-2xl border-border/80 bg-card/85 shadow-sm">
      <CardHeader className="space-y-3">
        <CardTitle className="text-3xl leading-tight sm:text-4xl lg:text-5xl">CLI Proxy API Dashboard</CardTitle>
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
          <Button className="h-11 w-full px-6 sm:w-auto" disabled={props.loading || !props.draftKey.trim()} type="submit">
            {props.loading ? <RefreshCw className="size-4 animate-spin" /> : null}
            {props.loading ? '连接中...' : '进入 Dashboard'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
