import { cn } from '@lib/utils/cn'

export function Spinner({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        'inline-block h-5 w-5 animate-spin rounded-full border-2 border-gray-300 border-t-primary-600',
        className,
      )}
      role="status"
      aria-label="Loading"
    />
  )
}
