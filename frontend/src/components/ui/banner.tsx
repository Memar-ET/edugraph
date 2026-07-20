import type { HTMLAttributes } from 'react'

import { cn } from '@lib/utils/cn'

const toneStyles = {
  error: 'bg-alert-50 text-alert-800 border-alert-200',
  success: 'bg-health-50 text-health-800 border-health-200',
  info: 'bg-primary-50 text-primary-800 border-primary-200',
  warning: 'bg-seal-50 text-seal-800 border-seal-200',
} as const

export interface BannerProps extends HTMLAttributes<HTMLDivElement> {
  tone?: keyof typeof toneStyles
}

export function Banner({ tone = 'info', className, ...props }: BannerProps) {
  return (
    <div
      role={tone === 'error' ? 'alert' : 'status'}
      className={cn('rounded-md border px-4 py-3 text-sm', toneStyles[tone], className)}
      {...props}
    />
  )
}
