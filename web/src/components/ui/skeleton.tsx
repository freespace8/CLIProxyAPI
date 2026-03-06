import { cn } from '@/lib/utils'

function Skeleton(props: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('animate-pulse rounded-md bg-muted/70', props.className)} {...props} />
}

export { Skeleton }
