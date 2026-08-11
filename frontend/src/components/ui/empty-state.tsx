import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'

import { cn } from '@lib/utils/cn'

export interface EmptyStateProps {
  icon?: LucideIcon
  title: string
  description?: string
  action?: ReactNode
  className?: string
}

export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-gray-300 bg-white px-6 py-12 text-center',
        className,
      )}
    >
      {Icon && <Icon className="h-8 w-8 text-gray-400" aria-hidden />}
      <p className="text-sm font-medium text-gray-900">{title}</p>
      {description && <p className="max-w-sm text-sm text-gray-500">{description}</p>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  )
}
