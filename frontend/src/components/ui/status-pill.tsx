import type { ReactNode } from 'react'

import { cn } from '@lib/utils/cn'

const toneStyles = {
  health: 'bg-health-50 text-health-700 ring-1 ring-inset ring-health-200',
  alert: 'bg-alert-50 text-alert-700 ring-1 ring-inset ring-alert-200',
  seal: 'bg-seal-50 text-seal-700 ring-1 ring-inset ring-seal-200',
  primary: 'bg-primary-50 text-primary-700 ring-1 ring-inset ring-primary-200',
  neutral: 'bg-gray-100 text-gray-700 ring-1 ring-inset ring-gray-200',
} as const

export interface StatusPillProps {
  tone?: keyof typeof toneStyles
  children: ReactNode
  className?: string
}

export function StatusPill({ tone = 'neutral', children, className }: StatusPillProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-medium',
        toneStyles[tone],
        className,
      )}
    >
      {children}
    </span>
  )
}

/** Shared mastery/coverage-percentage -> tone mapping so every dashboard
 * (subject health, heatmap, quality scores) uses the same thresholds. */
// eslint-disable-next-line react-refresh/only-export-components -- threshold helper belongs next to the tone map it derives from, not a component
export function toneForPct(pct: number): keyof typeof toneStyles {
  if (pct >= 70) return 'health'
  if (pct >= 40) return 'seal'
  return 'alert'
}
