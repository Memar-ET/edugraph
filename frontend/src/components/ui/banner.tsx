import type { HTMLAttributes } from 'react'

import { cn } from '@lib/utils/cn'

const toneStyles = {
  error: 'bg-red-50 text-red-800 border-red-200',
  success: 'bg-green-50 text-green-800 border-green-200',
  info: 'bg-blue-50 text-blue-800 border-blue-200',
  warning: 'bg-amber-50 text-amber-800 border-amber-200',
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
