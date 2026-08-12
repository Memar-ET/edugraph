import { useState } from 'react'
import type { LucideIcon } from 'lucide-react'
import { TrendingUp, TrendingDown, X } from 'lucide-react'
import { cn } from '@lib/utils/cn'
import { ThreeDCard } from '@components/ui'

export interface StatMetricCardProps {
  title: string
  value: string | number
  change?: string
  trend?: 'up' | 'down' | 'neutral'
  periodText?: string
  icon: LucideIcon
  detailsSubtitle?: string
  className?: string
}

export function StatMetricCard({
  title,
  value,
  change,
  trend = 'up',
  periodText = 'Last Month',
  icon: Icon,
  detailsSubtitle = 'Historical trend tracked against AI prerequisite benchmarks.',
  className,
}: StatMetricCardProps) {
  const [showDetailModal, setShowDetailModal] = useState(false)

  return (
    <>
      <ThreeDCard
        onClick={() => setShowDetailModal(true)}
        className={cn(
          'group cursor-pointer select-none border-gray-100/90 bg-white transition-all hover:border-gray-300',
          className,
        )}
      >
        <div className="flex items-start justify-between">
          <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-gray-100 text-gray-700 transition-all duration-300 group-hover:bg-gray-900 group-hover:text-white group-hover:scale-110 shadow-sm">
            <Icon className="h-5 w-5" aria-hidden="true" />
          </div>
          {change && (
            <div
              className={cn(
                'inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-semibold shadow-2xs transition-transform group-hover:scale-105',
                trend === 'up'
                  ? 'bg-emerald-50 text-emerald-700 border border-emerald-100/60'
                  : trend === 'down'
                    ? 'bg-rose-50 text-rose-700 border border-rose-100/60'
                    : 'bg-gray-100 text-gray-600',
              )}
            >
              {trend === 'up' ? (
                <TrendingUp className="h-3 w-3" />
              ) : trend === 'down' ? (
                <TrendingDown className="h-3 w-3" />
              ) : null}
              <span>{change}</span>
            </div>
          )}
        </div>

        <div className="mt-4">
          <p className="text-[11px] font-bold uppercase tracking-wider text-gray-400 group-hover:text-gray-600 transition-colors">
            {title}
          </p>
          <div className="mt-1 flex items-baseline justify-between">
            <h3 className="font-display text-2xl font-extrabold tracking-tight text-gray-900 lg:text-3xl">
              {value}
            </h3>
            {periodText && (
              <span className="text-xs font-semibold text-gray-400">{periodText}</span>
            )}
          </div>
        </div>
      </ThreeDCard>

      {/* Interactive Detail Modal Popover */}
      {showDetailModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-gray-900/40 backdrop-blur-sm p-4 animate-in fade-in">
          <div className="w-full max-w-md rounded-2xl border border-gray-100 bg-white p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between border-b border-gray-100 pb-3">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gray-900 text-white">
                  <Icon className="h-5 w-5" />
                </div>
                <div>
                  <h3 className="font-display text-lg font-bold text-gray-900">{title}</h3>
                  <p className="text-xs text-gray-500 font-medium">{periodText}</p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setShowDetailModal(false)}
                className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-900"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="space-y-3">
              <div className="rounded-xl bg-gray-50 p-4 border border-gray-100">
                <p className="text-xs text-gray-500 font-medium uppercase tracking-wider">Current Metric Value</p>
                <p className="font-display text-3xl font-extrabold text-gray-900 mt-1">{value}</p>
                {change && (
                  <p className="text-xs font-semibold text-emerald-600 mt-1">
                    {change} increase compared to previous assessment period
                  </p>
                )}
              </div>
              <p className="text-xs text-gray-600 leading-relaxed">
                {detailsSubtitle}
              </p>
            </div>

            <div className="flex justify-end pt-2">
              <button
                type="button"
                onClick={() => setShowDetailModal(false)}
                className="rounded-xl bg-gray-900 px-4 py-2 text-xs font-semibold text-white hover:bg-gray-800"
              >
                Close Insights
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
