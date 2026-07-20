import { AlertTriangle } from 'lucide-react'

import { cn } from '@lib/utils/cn'
import type { HeatmapTopic } from '@/types/api'

export interface HeatmapGridProps {
  topics: HeatmapTopic[]
  className?: string
}

function intensityTone(pct: number) {
  if (pct >= 50) return 'bg-alert-500'
  if (pct >= 25) return 'bg-seal-500'
  return 'bg-health-500'
}

/** Capability 4A class heatmap: topics ranked by share of the class struggling. */
export function HeatmapGrid({ topics, className }: HeatmapGridProps) {
  const sorted = [...topics].sort((a, b) => b.strugglingPct - a.strugglingPct)
  return (
    <div className={cn('divide-y divide-gray-100', className)}>
      {sorted.map((topic) => (
        <div key={topic.topicId} className="flex items-center gap-4 py-3">
          <div className="w-40 shrink-0 truncate text-sm font-medium text-gray-900 sm:w-48">{topic.title}</div>
          <div className="h-2.5 flex-1 overflow-hidden rounded-full bg-gray-100">
            <div
              className={cn('h-full rounded-full', intensityTone(topic.strugglingPct))}
              style={{ width: `${Math.min(100, topic.strugglingPct)}%` }}
            />
          </div>
          <div className="w-24 shrink-0 text-right font-mono text-xs text-gray-500">
            {Math.round(topic.strugglingPct)}% ({topic.studentsStruggling})
          </div>
          {topic.alert && (
            <span title={topic.alert.message}>
              <AlertTriangle className="h-4 w-4 shrink-0 text-alert-600" aria-label={topic.alert.message} />
            </span>
          )}
        </div>
      ))}
    </div>
  )
}
