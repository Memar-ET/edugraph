import { TrendingUp, TrendingDown, Minus } from 'lucide-react'

export interface KpiTrendCardProps {
  title: string
  value: string | number
  delta?: string
  deltaType?: 'increase' | 'decrease' | 'neutral'
  subtitle?: string
  sparklineData?: number[]
}

export function KpiTrendCard({
  title,
  value,
  delta,
  deltaType = 'increase',
  subtitle,
  sparklineData,
}: KpiTrendCardProps) {
  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm hover:border-slate-300 transition-all">
      <div className="flex items-center justify-between">
        <p className="text-xs font-semibold text-slate-500">{title}</p>
        {delta && (
          <div
            className={`flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-bold ${
              deltaType === 'increase'
                ? 'bg-emerald-100 text-emerald-800'
                : deltaType === 'decrease'
                ? 'bg-rose-100 text-rose-800'
                : 'bg-slate-100 text-slate-700'
            }`}
          >
            {deltaType === 'increase' ? (
              <TrendingUp className="h-3 w-3" />
            ) : deltaType === 'decrease' ? (
              <TrendingDown className="h-3 w-3" />
            ) : (
              <Minus className="h-3 w-3" />
            )}
            <span>{delta}</span>
          </div>
        )}
      </div>

      <div className="mt-2 flex items-baseline justify-between">
        <p className="text-2xl font-extrabold text-slate-900 font-display tracking-tight">{value}</p>
      </div>

      {subtitle && <p className="mt-1 text-[11px] text-slate-500 font-medium">{subtitle}</p>}

      {sparklineData && sparklineData.length > 1 && (
        <div className="mt-3 h-8 w-full">
          <svg className="h-full w-full" viewBox="0 0 100 30" preserveAspectRatio="none">
            <polyline
              fill="none"
              stroke="#0f766e"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              points={sparklineData
                .map((val, idx) => {
                  const min = Math.min(...sparklineData)
                  const max = Math.max(...sparklineData)
                  const range = max - min || 1
                  const x = (idx / (sparklineData.length - 1)) * 100
                  const y = 30 - ((val - min) / range) * 25
                  return `${x},${y}`
                })
                .join(' ')}
            />
          </svg>
        </div>
      )}
    </div>
  )
}
