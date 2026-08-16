import { useQuery } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useCallback, useEffect, useRef, useState } from 'react'

import { apiErrorMessage } from '@lib/api/client'
import { autosaveExamAnswers, getExamDraft, getMyExamInsight, listExamQuestions, submitExam } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, Spinner } from '@components/ui'
import { AppShell } from '@components/layout'
import type { ExamInsight, SubmitExamResponse } from '@/types/api'

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

type SaveStatus = 'idle' | 'saving' | 'saved' | 'error'

export function TakeExamPage() {
  const { examId } = useParams({ from: '/student/exams/$examId' })
  const navigate = useNavigate()

  const [answers, setAnswers] = useState<Record<string, string>>({})
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [result, setResult] = useState<SubmitExamResponse | null>(null)
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle')
  const [pollStarted, setPollStarted] = useState(false)
  const [pollTimedOut, setPollTimedOut] = useState(false)

  const idempotencyKey = useRef(getOrCreateIdempotencyKey(examId))
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const pollStartTimeRef = useRef<number | null>(null)
  const answersRef = useRef(answers)
  answersRef.current = answers

  const { data: questions, isLoading, error } = useQuery({
    queryKey: queryKeys.examQuestions(examId),
    queryFn: () => listExamQuestions(examId),
  })

  // Restore draft on mount
  useQuery({
    queryKey: queryKeys.examDraft(examId),
    queryFn: () => getExamDraft(examId),
    enabled: !result,
    retry: false,
    staleTime: Infinity,
    onSuccess: (draft) => {
      if (draft.answers.length > 0) {
        setAnswers((prev) => {
          const restored: Record<string, string> = {}
          draft.answers.forEach((a) => {
            restored[a.questionId] = a.response
          })
          return { ...restored, ...prev }
        })
      }
    },
  })

  // Insight polling after submit
  const insightQuery = useQuery({
    queryKey: queryKeys.myExamInsight(examId),
    queryFn: () => getMyExamInsight(examId),
    enabled: pollStarted && !pollTimedOut,
    refetchInterval: (data: ExamInsight | undefined) => {
      if (data) return false
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
    setSaveStatus('saving')
    try {
      await autosaveExamAnswers(
        examId,
        Object.entries(current).map(([questionId, response]) => ({ questionId, response })),
      )
      setSaveStatus('saved')
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

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [])

  const setAnswer = (questionId: string, response: string) => {
    setAnswers((prev) => ({ ...prev, [questionId]: response }))
    setSaveStatus('idle')
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      void doAutosave()
    }, AUTOSAVE_DEBOUNCE_MS)
  }

  const handleSubmit = async () => {
    if (!questions) return
    setSubmitError(null)
    setSubmitting(true)
    // Clear autosave intervals before submitting
    if (debounceRef.current) clearTimeout(debounceRef.current)
    if (intervalRef.current) clearInterval(intervalRef.current)
    try {
      const res = await submitExam(examId, {
        answers: questions.map((q) => ({ questionId: q.id, response: answers[q.id] ?? '' })),
        idempotencyKey: idempotencyKey.current,
      })
      setResult(res)
      setPollStarted(true)
      // Remove idempotency key from session so a retry generates a new one
      sessionStorage.removeItem(`exam-idempotency-${examId}`)
    } catch (err) {
      setSubmitError(apiErrorMessage(err, 'Submission failed.'))
    } finally {
      setSubmitting(false)
    }
  }

  const insight = insightQuery.data

  return (
    <AppShell title="Take exam">
      <div className="mx-auto max-w-2xl space-y-4">
        <div className="flex items-center justify-between">
          <Button variant="ghost" size="sm" onClick={() => void navigate({ to: '/student/exams' })}>
            ← Back
          </Button>
          {!result && (
            <span className="text-xs text-gray-400">
              {saveStatus === 'saving' && 'Saving…'}
              {saveStatus === 'saved' && '✓ Saved'}
              {saveStatus === 'error' && '⚠ Save failed'}
              {saveStatus === 'idle' && ''}
            </span>
          )}
        </div>

        {isLoading && (
          <div className="flex items-center gap-2 text-gray-500">
            <Spinner /> Loading exam…
          </div>
        )}
        {error && <Banner tone="error">{apiErrorMessage(error, 'Could not load this exam.')}</Banner>}

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
                      <div className="space-y-1.5">
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
