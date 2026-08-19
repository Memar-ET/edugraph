import { useCallback, useEffect, useRef } from 'react'

import { reportIntegrityEvents } from '@lib/api/endpoints'
import type { IntegrityEventInput, IntegrityEventType } from '@/types/api'

const FLUSH_INTERVAL_MS = 10_000

/** Tab-visibility/fullscreen/connection signals for the active exam
 * attempt -- none of this existed anywhere in the app before. These are
 * diagnostic/troubleshooting signals, per Part 20's explicit framing,
 * never automatic proof of misconduct -- the teacher-facing summary
 * shows plain counts ("N visibility changes"), not accusations, and
 * nothing here blocks or alters the student's exam.
 *
 * Batched and flushed periodically rather than one request per event --
 * a burst of rapid tab-switches must not flood the autosave/submit
 * requests that actually matter. A failed flush re-queues its batch
 * rather than dropping it, but this is still best-effort: a permanently
 * lost batch (e.g. the tab closes before the next flush) is an accepted
 * gap, not a correctness issue, since it never gates grading. */
export function useIntegrityEvents(record: (type: IntegrityEventType, metadata?: Record<string, unknown>) => void) {
  useEffect(() => {
    const onVisibility = () => record(document.hidden ? 'tab_hidden' : 'tab_visible')
    const onFullscreenChange = () => record(document.fullscreenElement ? 'fullscreen_entered' : 'fullscreen_exited')
    document.addEventListener('visibilitychange', onVisibility)
    document.addEventListener('fullscreenchange', onFullscreenChange)
    return () => {
      document.removeEventListener('visibilitychange', onVisibility)
      document.removeEventListener('fullscreenchange', onFullscreenChange)
    }
  }, [record])
}

/** Owns the event queue + periodic flush; returns the `record` callback
 * useIntegrityEvents (and the caller's own online/offline handling)
 * feed into. Split from useIntegrityEvents so the queue/flush logic is
 * reusable independent of which browser events feed it. */
export function useIntegrityEventQueue(examId: string, enabled: boolean) {
  const seqRef = useRef(0)
  const queueRef = useRef<IntegrityEventInput[]>([])

  const record = useCallback(
    (eventType: IntegrityEventType, metadata?: Record<string, unknown>) => {
      if (!enabled) return
      queueRef.current.push({
        eventType,
        occurredAt: new Date().toISOString(),
        sequenceNumber: seqRef.current++,
        metadata,
      })
    },
    [enabled],
  )

  useEffect(() => {
    if (!enabled) return
    const interval = setInterval(() => {
      if (queueRef.current.length === 0) return
      const batch = queueRef.current
      queueRef.current = []
      reportIntegrityEvents(examId, batch).catch(() => {
        // Best-effort retry: put the failed batch back at the front so
        // it's included in the next flush instead of silently dropped.
        queueRef.current = [...batch, ...queueRef.current]
      })
    }, FLUSH_INTERVAL_MS)
    return () => clearInterval(interval)
  }, [enabled, examId])

  return record
}
