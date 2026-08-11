import { cn } from '@lib/utils/cn'

// The signature element of the design system: an ink-stamp badge for the
// platform's real ratification moments -- an approved curriculum job, a
// published exam, a certified quality score. Not decoration; it marks the
// same action a ministry official's rubber stamp marks on paper today.
const toneStyles = {
  seal: 'border-seal-600 text-seal-700 bg-seal-50',
  health: 'border-health-600 text-health-700 bg-health-50',
  alert: 'border-alert-600 text-alert-700 bg-alert-50',
} as const

export interface SealProps {
  label: string
  meta?: string
  tone?: keyof typeof toneStyles
  className?: string
}

export function Seal({ label, meta, tone = 'seal', className }: SealProps) {
  return (
    <div
      role="img"
      aria-label={meta ? `${label}, ${meta}` : label}
      className={cn(
        'inline-flex h-20 w-20 shrink-0 -rotate-6 flex-col items-center justify-center rounded-full border-4 border-double text-center font-mono leading-tight',
        toneStyles[tone],
        className,
      )}
    >
      <span className="px-2 text-[9px] font-semibold uppercase tracking-widest">{label}</span>
      {meta && <span className="mt-0.5 px-2 text-[8px] uppercase tracking-wide opacity-80">{meta}</span>}
    </div>
  )
}
