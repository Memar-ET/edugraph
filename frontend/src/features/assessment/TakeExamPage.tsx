import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useCallback, useEffect, useRef, useState } from 'react'

import { apiErrorMessage } from '@lib/api/client'
import { autosaveExamAnswers, getExamDraft, getMyExamInsight, startExamAttempt, submitExam } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useOnlineStatus } from '@lib/utils/useOnlineStatus'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, Spinner } from '@components/ui'
import { AppShell } from '@components/layout'
import { ExamTimer } from '@components/exam/ExamTimer'
import type { ExamInsight, SubmitExamResponse } from '@/types/api'
import { useIntegrityEventQueue, useIntegrityEvents } from './useIntegrityEvents'

const MCQ_OPTIONS = ['A', 'B', 'C', 'D'] as const
const AUTOSAVE_DEBOUNCE_MS = 2_000
const AUTOSAVE_INTERVAL_MS = 30_000
const INSIGHT_POLL_MS = 5_000
const INSIGHT_POLL_TIMEOUT_MS = 120_000

function getOrCreateIdempotencyKey(examId: string): string {
  const key = `exam-idempotency-${examId}`
  const existing = sessionStorage.getItem(key)
  if (existing) return existing
  const uuid = crypto.randomUUID()
  sessionStorage.setItem(key, uuid)
  return uuid
}

type SaveStatus = 'idle' | 'saving' | 'saved' | 'error' | 'offline'

export function TakeExamPage() {
  const { examId } = useParams({ from: '/student/exams/$examId' })
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const isOnline = useOnlineStatus()

  const [answers, setAnswers] = useState<Record<string, string>>({})
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [result, setResult] = useState<SubmitExamResponse | null>(null)
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle')
  const [lastSavedAt, setLastSavedAt] = useState<Date | null>(null)
  const [pollStarted, setPollStarted] = useState(false)
  const [pollTimedOut, setPollTimedOut] = useState(false)

  const idempotencyKey = useRef(getOrCreateIdempotencyKey(examId))
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const pollStartTimeRef = useRef<number | null>(null)
  const answersRef = useRef(answers)
  answersRef.current = answers
  const submittingRef = useRef(false)

  // The exam session itself -- creates (or resumes, if one is already
  // open) the server-authoritative attempt. Everything else on this page
  // (question order, option order, timer) comes from this response, not
  // re-derived client-side. Safe to treat as a query (not a one-shot
  // mutation): StartAttempt is idempotent while in_progress, so a
  // refetch (e.g. React StrictMode's double-invoke, or a stale-query
  // refetch) resumes the same session rather than creating a new one.
  const attemptQuery = useQuery({
    queryKey: queryKeys.examQuestions(examId),
    queryFn: () => startExamAttempt(examId),
    enabled: !result,
    retry: false,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  })
  const questions = attemptQuery.data?.questions

  const record = useIntegrityEventQueue(examId, Boolean(attemptQuery.data) && !result)
  useIntegrityEvents(record)

  const wasOnlineRef = useRef(isOnline)
  useEffect(() => {
    if (wasOnlineRef.current !== isOnline) {
      record(isOnline ? 'connection_restored' : 'connection_lost')
      wasOnlineRef.current = isOnline
    }
  }, [isOnline, record])

  // Restore draft on mount
  const draftQuery = useQuery({
    queryKey: queryKeys.examDraft(examId),
    queryFn: () => getExamDraft(examId),
    enabled: Boolean(attemptQuery.data) && !result,
    retry: false,
    staleTime: Infinity,
  })

  useEffect(() => {
    const draft = draftQuery.data
    if (draft && draft.answers.length > 0) {
      setAnswers((prev) => {
        const restored: Record<string, string> = {}
        draft.answers.forEach((a) => {
          restored[a.questionId] = a.response
        })
        return { ...restored, ...prev }
      })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draftQuery.data])

  // Insight polling after submit
  const insightQuery = useQuery<ExamInsight>({
    queryKey: queryKeys.myExamInsight(examId),
    queryFn: () => getMyExamInsight(examId),
    enabled: pollStarted && !pollTimedOut,
    refetchInterval: (query) => {
      if (query.state.data) return false
      if (pollTimedOut) return false
      return INSIGHT_POLL_MS
    },
    retry: false,
  })

  useEffect(() => {
    if (!pollStarted) return
    pollStartTimeRef.current = Date.now()
    const timeout = setTimeout(() => {
      setPollTimedOut(true)
    }, INSIGHT_POLL_TIMEOUT_MS)
    return () => clearTimeout(timeout)
  }, [pollStarted])

  const doAutosave = useCallback(async () => {
    const current = answersRef.current
    if (Object.keys(current).length === 0) return
    if (!navigator.onLine) {
      setSaveStatus('offline')
      return
    }
    setSaveStatus('saving')
    try {
      await autosaveExamAnswers(
        examId,
        Object.entries(current).map(([questionId, response]) => ({ questionId, response })),
      )
      setSaveStatus('saved')
      setLastSavedAt(new Date())
    } catch {
      setSaveStatus('error')
    }
  }, [examId])

  // 30s interval autosave
  useEffect(() => {
    if (result) return
    intervalRef.current = setInterval(() => {
      void doAutosave()
    }, AUTOSAVE_INTERVAL_MS)
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [doAutosave, result])

  // A dropped connection can strand unsaved answers -- retry once it
  // comes back rather than waiting for the next 30s tick.
  useEffect(() => {
    if (isOnline && saveStatus === 'offline') {
      void doAutosave()
    }
  }, [isOnline, saveStatus, doAutosave])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [])

  // Warn before an accidental tab-close/navigation away mid-exam --
  // answers are autosaved, but the student should still have to
  // confirm, not lose their place by a stray back-button tap.
  useEffect(() => {
    if (result) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
      e.returnValue = ''
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [result])

  const setAnswer = (questionId: string, response: string) => {
    setAnswers((prev) => ({ ...prev, [questionId]: response }))
    setSaveStatus('idle')
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      void doAutosave()
    }, AUTOSAVE_DEBOUNCE_MS)
  }

  const handleSubmit = useCallback(async () => {
    if (!questions || submittingRef.current) return
    submittingRef.current = true
    setSubmitError(null)
    setSubmitting(true)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    if (intervalRef.current) clearInterval(intervalRef.current)
    try {
      const res = await submitExam(examId, {
        answers: questions.map((q) => ({ questionId: q.id, response: answers[q.id] ?? '' })),
        idempotencyKey: idempotencyKey.current,
      })
      setResult(res)
      setPollStarted(true)
      sessionStorage.removeItem(`exam-idempotency-${examId}`)
      void queryClient.invalidateQueries({ queryKey: queryKeys.examQuestions(examId) })
    } catch (err) {
      setSubmitError(apiErrorMessage(err, 'Submission failed.'))
    } finally {
      setSubmitting(false)
      submittingRef.current = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [questions, answers, examId, queryClient])

  // Server time, not the client clock, is what actually enforces
  // expiry -- this local countdown reaching zero is only a UI trigger to
  // attempt submission promptly. If this request happens to land after
  // expires_at, the server still accepts and grades it (tagged
  // time_expired instead of student_submit) rather than rejecting it;
  // if the auto-submit ticker already finalized the attempt first, this
  // call idempotently returns that same result instead of erroring.
  const handleExpire = useCallback(() => {
    void handleSubmit()
  }, [handleSubmit])

  const insight = insightQuery.data

  const saveStatusLabel = !isOnline
    ? 'Offline — will retry when reconnected'
    : saveStatus === 'saving'
      ? 'Saving…'
      : saveStatus === 'saved'
        ? lastSavedAt
          ? `Saved at ${lastSavedAt.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}`
          : 'Saved'
        : saveStatus === 'error'
          ? 'Save failed — retrying…'
          : saveStatus === 'offline'
            ? 'Offline — will retry when reconnected'
            : ''

  return (
    <AppShell title="Take exam">
      <div className="mx-auto max-w-2xl space-y-4">
        <div className="flex items-center justify-between gap-3">
          <Button variant="ghost" size="sm" onClick={() => void navigate({ to: '/student/exams' })}>
            ← Back
          </Button>
          <div className="flex items-center gap-3">
            {!result && !isOnline && (
              <span className="rounded-full bg-alert-100 px-2.5 py-0.5 text-xs font-medium text-alert-800">
                Connection lost
              </span>
            )}
            {!result && saveStatusLabel && (
              <span className="text-xs text-gray-400" role="status">
                {saveStatusLabel}
              </span>
            )}
            {!result && attemptQuery.data && <ExamTimer expiresAt={attemptQuery.data.expiresAt} onExpire={handleExpire} />}
          </div>
        </div>

        {attemptQuery.isLoading && (
          <div className="flex items-center gap-2 text-gray-500">
            <Spinner /> Starting exam…
          </div>
        )}
        {attemptQuery.isError && (
          <Banner tone="error">{apiErrorMessage(attemptQuery.error, 'Could not start this exam.')}</Banner>
        )}

        {result && (
          <Card>
            <CardHeader>
              <CardTitle>Submitted</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <p>
                {result.gradedCount} of {result.gradedCount + result.pendingGradingCount} question(s)
                graded instantly.
              </p>
              {result.pendingGradingCount > 0 && (
                <Banner tone="info">
                  {result.pendingGradingCount} question(s) need your teacher to grade them before your
                  final score is ready.
                </Banner>
              )}
              {result.percentage !== undefined && (
                <Banner tone={result.passed ? 'success' : 'error'}>
                  Score: {result.totalScore} ({result.percentage.toFixed(1)}%) —{' '}
                  {result.passed ? 'Passed' : 'Not passed'}
                </Banner>
              )}

              {/* Insight polling status */}
              {pollStarted && !insight && !pollTimedOut && (
                <div className="flex items-center gap-2 rounded-lg bg-indigo-50 px-4 py-3 text-xs text-indigo-700">
                  <Spinner />
                  <span>Analysing your gaps — this takes a moment…</span>
                </div>
              )}
              {pollTimedOut && !insight && (
                <Banner tone="info">
                  Gap analysis is still running. Check back in a few minutes under your exam results.
                </Banner>
              )}
              {insight && (
                <div className="rounded-xl border border-indigo-100 bg-indigo-50 p-4 space-y-2">
                  <p className="text-sm font-semibold text-indigo-900">Gap Analysis Ready</p>
                  {insight.summary && (
                    <p className="text-xs text-indigo-700 leading-relaxed">{insight.summary}</p>
                  )}
                  <p className="text-xs text-indigo-600">
                    {insight.gapsFound} gap{insight.gapsFound !== 1 ? 's' : ''} identified.
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {questions && !result && (
          <>
            {submitError && <Banner tone="error">{submitError}</Banner>}
            <div className="space-y-4">
              {questions.map((q) => (
                <Card key={q.id}>
                  <CardContent className="space-y-3 pt-6">
                    {q.partLabel && <p className="text-xs font-medium text-gray-400">{q.partLabel}</p>}
                    <p className="text-sm font-medium text-gray-900">
                      {q.sequenceNumber}. {q.questionText}{' '}
                      <span className="font-normal text-gray-400">
                        ({q.marks} mark{q.marks === 1 ? '' : 's'})
                      </span>
                    </p>

                    {q.questionType === 'mcq' && q.options && q.options.length > 0 ? (
                      <div className="space-y-1.5" role="radiogroup" aria-label={`Question ${q.sequenceNumber} options`}>
                        {q.options.map((opt) => (
                          <label
                            key={opt.letter}
                            className={`flex cursor-pointer items-start gap-2 rounded-md border p-2 text-sm ${
                              answers[q.id] === opt.letter
                                ? 'border-blue-400 bg-blue-50'
                                : 'border-gray-200 hover:bg-gray-50'
                            }`}
                          >
                            <input
                              type="radio"
                              name={q.id}
                              value={opt.letter}
                              checked={answers[q.id] === opt.letter}
                              onChange={() => setAnswer(q.id, opt.letter)}
                              className="mt-0.5"
                            />
                            <span>
                              <span className="font-medium">{opt.letter})</span> {opt.text}
                            </span>
                          </label>
                        ))}
                      </div>
                    ) : q.questionType === 'mcq' ? (
                      <div className="flex gap-3">
                        {MCQ_OPTIONS.map((opt) => (
                          <label key={opt} className="flex items-center gap-1.5 text-sm">
                            <input
                              type="radio"
                              name={q.id}
                              value={opt}
                              checked={answers[q.id] === opt}
                              onChange={() => setAnswer(q.id, opt)}
                            />
                            {opt}
                          </label>
                        ))}
                      </div>
                    ) : (
                      <textarea
                        className="w-full rounded-md border border-gray-300 p-2 text-sm"
                        rows={3}
                        value={answers[q.id] ?? ''}
                        onChange={(e) => setAnswer(q.id, e.target.value)}
                      />
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>

            <div className="sticky bottom-4 flex justify-end">
              <Button size="lg" isLoading={submitting} onClick={() => void handleSubmit()}>
                Submit exam
              </Button>
            </div>
          </>
        )}
      </div>
    </AppShell>
  )
}
