import { useEffect, useRef, useState } from 'react'

import { cn } from '@lib/utils/cn'

const WARNING_THRESHOLD_SECS = 5 * 60
const CRITICAL_THRESHOLD_SECS = 60

function formatDuration(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  const mm = String(m).padStart(2, '0')
  const ss = String(sec).padStart(2, '0')
  return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`
}

export interface ExamTimerProps {
  /** ISO timestamp the server computed at attempt-start. Undefined means
   * this exam has no time limit. */
  expiresAt?: string
  /** Fired once, client-side, when the local countdown reaches zero --
   * a UI cue to trigger auto-submit. The server's own expires_at check
   * on every autosave/submit request is the real enforcement; this is
   * never trusted as the source of truth (see Part 41: never trust the
   * client's clock). */
  onExpire?: () => void
  className?: string
}

/** Server-authoritative exam countdown: expiresAt is a fixed point in
 * time set once by the server at StartAttempt, never recomputed or
 * extended client-side. Manipulating the local system clock only skews
 * the DISPLAY -- every autosave/submit request is independently checked
 * against the server's own clock and rejected if expired regardless of
 * what this component shows. */
export function ExamTimer({ expiresAt, onExpire, className }: ExamTimerProps) {
  const [remainingSecs, setRemainingSecs] = useState<number | null>(() =>
    expiresAt ? (new Date(expiresAt).getTime() - Date.now()) / 1000 : null,
  )
  const firedRef = useRef(false)

  useEffect(() => {
    if (!expiresAt) {
      setRemainingSecs(null)
      return
    }
    firedRef.current = false
    const expiryMs = new Date(expiresAt).getTime()
    const tick = () => {
      const remaining = (expiryMs - Date.now()) / 1000
      setRemainingSecs(remaining)
      if (remaining <= 0 && !firedRef.current) {
        firedRef.current = true
        onExpire?.()
      }
    }
    tick()
    const interval = setInterval(tick, 1000)
    return () => clearInterval(interval)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onExpire intentionally not a dep: re-subscribing on every render identity change would reset firedRef's single-fire guarantee
  }, [expiresAt])

  if (!expiresAt || remainingSecs === null) {
    return (
      <span className={cn('text-xs font-medium text-gray-400', className)} aria-label="No time limit">
        No time limit
      </span>
    )
  }

  const isCritical = remainingSecs <= CRITICAL_THRESHOLD_SECS
  const isWarning = remainingSecs <= WARNING_THRESHOLD_SECS

  return (
    <span
      role="timer"
      aria-live={isWarning ? 'assertive' : 'polite'}
      aria-label={`Time remaining: ${formatDuration(remainingSecs)}`}
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-sm font-semibold tabular-nums',
        isCritical
          ? 'animate-pulse bg-alert-100 text-alert-800'
          : isWarning
            ? 'bg-seal-100 text-seal-800'
            : 'bg-gray-100 text-gray-700',
        className,
      )}
    >
      {formatDuration(remainingSecs)}
      {isWarning && <span className="sr-only"> remaining — time is running low</span>}
    </span>
  )
}
