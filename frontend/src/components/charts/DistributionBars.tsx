import { cn } from '@lib/utils/cn'

export interface DistributionBarsProps {
  /** label -> percentage (0-100). */
  data: Record<string, number>
  tone?: 'primary' | 'health' | 'seal' | 'alert'
  className?: string
}

const TONE_BAR = {
  primary: 'bg-primary-500',
  health: 'bg-health-500',
  seal: 'bg-seal-500',
  alert: 'bg-alert-500',
} as const

/** Horizontal percentage-bar breakdown, e.g. Bloom-level or difficulty distribution. */
export function DistributionBars({ data, tone = 'primary', className }: DistributionBarsProps) {
  const entries = Object.entries(data)
  return (
    <div className={cn('space-y-2', className)}>
      {entries.map(([label, pct]) => (
        <div key={label} className="flex items-center gap-3 text-sm">
          <span className="w-32 shrink-0 truncate capitalize text-gray-600">{label.replace(/_/g, ' ')}</span>
          <div className="h-2 flex-1 overflow-hidden rounded-full bg-gray-100">
            <div
              className={cn('h-full rounded-full', TONE_BAR[tone])}
              style={{ width: `${Math.min(100, Math.max(0, pct))}%` }}
            />
          </div>
          <span className="w-10 shrink-0 text-right font-mono text-xs text-gray-500">{Math.round(pct)}%</span>
        </div>
      ))}
    </div>
  )
}
