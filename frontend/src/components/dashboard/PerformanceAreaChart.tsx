import { useState } from 'react'
import {
  Area,
  AreaChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { ChevronDown, Sparkles } from 'lucide-react'
import { cn } from '@lib/utils/cn'
import { ThreeDCard } from '@components/ui'

export interface ChartDataPoint {
  label: string
  value: number
  previous?: number
}

export interface PerformanceAreaChartProps {
  title?: string
  subtitle?: string
  data: ChartDataPoint[]
  className?: string
  unit?: string
}

const WEEKLY_DATA: ChartDataPoint[] = [
  { label: 'W1', value: 72 },
  { label: 'W2', value: 78 },
  { label: 'W3', value: 84 },
  { label: 'W4', value: 91 },
]

export function PerformanceAreaChart({
  title = 'Students Performance Statistics',
  subtitle,
  data,
  className,
  unit = '%',
}: PerformanceAreaChartProps) {
  const [period, setPeriod] = useState<'Monthly' | 'Weekly'>('Monthly')
  const [activeData, setActiveData] = useState<ChartDataPoint[]>(data)

  const togglePeriod = () => {
    if (period === 'Monthly') {
      setPeriod('Weekly')
      setActiveData(WEEKLY_DATA)
    } else {
      setPeriod('Monthly')
      setActiveData(data)
    }
  }

  return (
    <ThreeDCard
      className={cn('flex flex-col border-gray-100/90 bg-white p-5', className)}
    >
      <div className="flex items-center justify-between pb-4">
        <div>
          <h3 className="font-display text-lg font-bold tracking-tight text-gray-900 flex items-center gap-2">
            <span>{title}</span>
            <Sparkles className="h-4 w-4 text-amber-500" />
          </h3>
          {subtitle && <p className="text-xs font-medium text-gray-500">{subtitle}</p>}
        </div>
        <div className="relative">
          <button
            type="button"
            onClick={togglePeriod}
            className="flex items-center gap-1.5 rounded-xl border border-gray-200 bg-gray-50 px-3 py-1.5 text-xs font-semibold text-gray-700 shadow-2xs transition-all hover:bg-gray-900 hover:text-white"
          >
            <span>{period}</span>
            <ChevronDown className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      <div className="h-64 w-full pt-2">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={activeData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
            <defs>
              <linearGradient id="performanceGradient3D" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#1e1f22" stopOpacity={0.25} />
                <stop offset="95%" stopColor="#1e1f22" stopOpacity={0.0} />
              </linearGradient>
            </defs>
            <XAxis
              dataKey="label"
              axisLine={false}
              tickLine={false}
              tick={{ fill: '#6b7280', fontSize: 12, fontWeight: 600 }}
              dy={10}
            />
            <YAxis
              axisLine={false}
              tickLine={false}
              tick={{ fill: '#9ca3af', fontSize: 12 }}
            />
            <Tooltip
              content={({ active, payload }) => {
                if (active && payload && payload.length && payload[0]) {
                  return (
                    <div className="rounded-xl bg-gray-900 px-3 py-2 text-xs font-bold text-white shadow-2xl border border-gray-800">
                      <p>{`${payload[0].value}${unit}`}</p>
                    </div>
                  )
                }
                return null
              }}
            />
            <Area
              type="monotone"
              dataKey="value"
              stroke="#1e1f22"
              strokeWidth={3.5}
              fillOpacity={1}
              fill="url(#performanceGradient3D)"
              activeDot={{ r: 7, fill: '#1e1f22', stroke: '#ffffff', strokeWidth: 3 }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </ThreeDCard>
  )
}
