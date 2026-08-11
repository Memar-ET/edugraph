import { useState } from 'react'
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'
import { Calendar } from 'lucide-react'
import { cn } from '@lib/utils/cn'
import { ThreeDCard } from '@components/ui'

export interface DonutSegment {
  name: string
  value: number
  color: string
}

export interface DistributionDonutChartProps {
  title?: string
  totalValue?: string | number
  centerLabel?: string
  centerPercentage?: string | number
  segments: DonutSegment[]
  dateLabel?: string
  className?: string
}

export function DistributionDonutChart({
  title = 'Mastery & Gap Overview',
  totalValue = '88% Score',
  centerLabel = 'Mastery',
  centerPercentage = '88%',
  segments,
  dateLabel = 'Current Term',
  className,
}: DistributionDonutChartProps) {
  const [activeSegment, setActiveSegment] = useState<DonutSegment | null>(null)
  const [currentDateLabel, setCurrentDateLabel] = useState(dateLabel)

  const toggleDateFilter = () => {
    setCurrentDateLabel((prev) => (prev === 'Current Term' ? 'Prior Term' : 'Current Term'))
  }

  return (
    <ThreeDCard
      className={cn('flex flex-col border-gray-100/90 bg-white p-5', className)}
    >
      <div className="flex items-center justify-between pb-2">
        <h3 className="font-display text-lg font-bold tracking-tight text-gray-900">{title}</h3>
        <button
          type="button"
          onClick={toggleDateFilter}
          className="flex items-center gap-1.5 rounded-xl border border-gray-200 bg-gray-50 px-2.5 py-1 text-xs font-semibold text-gray-600 hover:bg-gray-900 hover:text-white transition-all"
        >
          <Calendar className="h-3.5 w-3.5" />
          <span>{currentDateLabel}</span>
        </button>
      </div>

      {totalValue && (
        <div className="mt-1">
          <p className="text-[10px] text-gray-400 font-bold uppercase tracking-wider">Total Volume / Metric</p>
          <p className="font-display text-xl font-extrabold text-gray-900">{totalValue}</p>
        </div>
      )}

      {/* Donut Chart with Interactive Center Percentage */}
      <div className="relative my-2 flex h-52 items-center justify-center">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Tooltip
              content={({ active, payload }) => {
                if (active && payload && payload.length && payload[0]) {
                  const data = payload[0].payload as DonutSegment
                  return (
                    <div className="rounded-xl bg-gray-900 px-3 py-1.5 text-xs font-bold text-white shadow-xl">
                      {data.name}: {data.value}%
                    </div>
                  )
                }
                return null
              }}
            />
            <Pie
              data={segments}
              cx="50%"
              cy="50%"
              innerRadius={60}
              outerRadius={82}
              paddingAngle={4}
              dataKey="value"
              stroke="none"
              onMouseEnter={(_, index) => setActiveSegment(segments[index] ?? null)}
              onMouseLeave={() => setActiveSegment(null)}
            >
              {segments.map((entry, index) => (
                <Cell
                  key={`cell-${index}`}
                  fill={entry.color}
                  className="transition-transform duration-300 hover:scale-105 cursor-pointer"
                />
              ))}
            </Pie>
          </PieChart>
        </ResponsiveContainer>

        {/* Center Pill Display */}
        <div className="pointer-events-none absolute flex flex-col items-center justify-center text-center">
          <span className="font-display text-2xl font-black tracking-tight text-gray-900">
            {activeSegment ? `${activeSegment.value}%` : centerPercentage}
          </span>
          <span className="text-[10px] font-bold uppercase tracking-wider text-gray-400">
            {activeSegment ? activeSegment.name : centerLabel}
          </span>
        </div>
      </div>

      {/* Legend Grid */}
      <div className="mt-2 grid grid-cols-2 gap-2 border-t border-gray-100 pt-3 text-xs">
        {segments.map((seg) => (
          <div
            key={seg.name}
            onMouseEnter={() => setActiveSegment(seg)}
            onMouseLeave={() => setActiveSegment(null)}
            className={cn(
              'flex items-center gap-2 rounded-lg p-1 transition-colors cursor-pointer',
              activeSegment?.name === seg.name && 'bg-gray-100 font-bold',
            )}
          >
            <span
              className="h-2.5 w-2.5 rounded-full shadow-xs"
              style={{ backgroundColor: seg.color }}
            />
            <span className="truncate text-gray-600 font-semibold">{seg.name}</span>
            <span className="ml-auto font-bold text-gray-900">{seg.value}%</span>
          </div>
        ))}
      </div>
    </ThreeDCard>
  )
}
