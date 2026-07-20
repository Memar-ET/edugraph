import { useQuery } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useState } from 'react'

import { apiErrorMessage } from '@lib/api/client'
import { listExamQuestions, submitExam } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, Spinner } from '@components/ui'
import { AppShell } from '@components/layout'
import type { SubmitExamResponse } from '@/types/api'

const MCQ_OPTIONS = ['A', 'B', 'C', 'D'] as const

export function TakeExamPage() {
  const { examId } = useParams({ from: '/student/exams/$examId' })
  const navigate = useNavigate()

  const [answers, setAnswers] = useState<Record<string, string>>({})
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [result, setResult] = useState<SubmitExamResponse | null>(null)

  const { data: questions, isLoading, error } = useQuery({
    queryKey: queryKeys.examQuestions(examId),
    queryFn: () => listExamQuestions(examId),
  })

  const setAnswer = (questionId: string, response: string) => {
    setAnswers((prev) => ({ ...prev, [questionId]: response }))
  }

  const handleSubmit = async () => {
    if (!questions) return
    setSubmitError(null)
    setSubmitting(true)
    try {
      const res = await submitExam(examId, {
        answers: questions.map((q) => ({ questionId: q.id, response: answers[q.id] ?? '' })),
      })
      setResult(res)
    } catch (err) {
      setSubmitError(apiErrorMessage(err, 'Submission failed.'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AppShell title="Take exam">
      <div className="mx-auto max-w-2xl space-y-4">
        <Button variant="ghost" size="sm" onClick={() => void navigate({ to: '/student/exams' })}>
          ← Back
        </Button>

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
            <CardContent className="space-y-2 text-sm">
              <p>
                {result.gradedCount} of {result.gradedCount + result.pendingGradingCount} question(s)
                graded instantly.
              </p>
              {result.pendingGradingCount > 0 && (
                <Banner tone="info">
                  {result.pendingGradingCount} question(s) need your teacher to grade them by hand before
                  your final score is ready.
                </Banner>
              )}
              {result.percentage !== undefined && (
                <Banner tone={result.passed ? 'success' : 'error'}>
                  Score: {result.totalScore} ({result.percentage.toFixed(1)}%) —{' '}
                  {result.passed ? 'Passed' : 'Not passed'}
                </Banner>
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
                      // Older exams parsed before option-splitting existed --
                      // options are still embedded in questionText above, so
                      // fall back to plain lettered radios.
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
