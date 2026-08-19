/** A tiny inline sparkline showing a short numeric series.
 *  Uses SVG polyline, no recharts dependency for this lightweight use-case.
 */
interface SparklineMiniProps {
  data: number[]
  color?: string
  width?: number
  height?: number
  label?: string
}

export function SparklineMini({
  data,
  color = '#1D3450',
  width = 64,
  height = 24,
  label,
}: SparklineMiniProps) {
  if (!data || data.length < 2) return null

  const min = Math.min(...data)
  const max = Math.max(...data)
  const range = max - min || 1

  const points = data
    .map((v, i) => {
      const x = (i / (data.length - 1)) * width
      const y = height - ((v - min) / range) * height
      return `${x},${y}`
    })
    .join(' ')

  const trend = data[data.length - 1]! - data[0]!
  const trendLabel = trend > 0 ? 'improving' : trend < 0 ? 'declining' : 'stable'

  return (
    <span aria-label={label ?? `Trend: ${trendLabel}`} role="img">
      <svg
        width={width}
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        aria-hidden
        className="inline-block align-middle"
      >
        <polyline
          points={points}
          fill="none"
          stroke={color}
          strokeWidth={1.5}
          strokeLinejoin="round"
          strokeLinecap="round"
        />
      </svg>
    </span>
  )
}
