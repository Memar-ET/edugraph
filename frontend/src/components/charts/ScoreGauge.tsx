import { cn } from '@lib/utils/cn'

export interface ScoreGaugeProps {
  /** 0-100. */
  value: number
  label?: string
  size?: number
  tone?: 'health' | 'seal' | 'alert' | 'primary'
  className?: string
}

const TONE_STROKE = {
  health: '#2F6844',
  seal: '#B8863B',
  alert: '#A23E2A',
  primary: '#284769',
} as const

/** Single-value radial gauge used for composite quality/mastery scores. */
export function ScoreGauge({ value, label, size = 96, tone = 'primary', className }: ScoreGaugeProps) {
  const clamped = Math.max(0, Math.min(100, value))
  const radius = (size - 12) / 2
  const circumference = 2 * Math.PI * radius
  const offset = circumference * (1 - clamped / 100)

  return (
    <div className={cn('inline-flex flex-col items-center gap-1', className)}>
      <svg
        width={size}
        height={size}
        viewBox={`0 0 ${size} ${size}`}
        role="img"
        aria-label={`${label ?? 'Score'}: ${Math.round(clamped)} out of 100`}
      >
        <circle cx={size / 2} cy={size / 2} r={radius} fill="none" stroke="#E1DACA" strokeWidth={8} />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke={TONE_STROKE[tone]}
          strokeWidth={8}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          transform={`rotate(-90 ${size / 2} ${size / 2})`}
        />
        <text
          x="50%"
          y="50%"
          textAnchor="middle"
          dominantBaseline="central"
          className="fill-gray-900 font-mono text-xl font-semibold"
        >
          {Math.round(clamped)}
        </text>
      </svg>
      {label && <span className="text-center text-xs font-medium text-gray-600">{label}</span>}
    </div>
  )
}
