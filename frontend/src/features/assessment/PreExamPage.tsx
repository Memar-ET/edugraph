import { useNavigate, useParams } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { AlertCircle, BookOpen, Clock, FileText, Info, RotateCcw } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, Spinner } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { getExam } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

const SCOPE_LABELS: Record<string, string> = {
  unit_test: 'Unit Test',
  midterm: 'Midterm',
  final_exam: 'Final Exam',
}

export function PreExamPage() {
  const { examId } = useParams({ from: '/student/exams/$examId/pre' })
  const navigate = useNavigate()
  const [understood, setUnderstood] = useState(false)

  const { data: exam, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.exam(examId),
    queryFn: () => getExam(examId),
  })

  // The actual server-side attempt (assessment.exam_attempts row) is
  // created by TakeExamPage's own call to POST /start on mount, not
  // here -- this page is purely informational. Requiring the checkbox
  // before this navigation is what makes "attempt creation must not
  // happen just from viewing the info page" concrete: without it, a
  // student can view duration/instructions/attempts-allowed without
  // ever starting a real timed session.
  const startExam = () => {
    void navigate({ to: '/student/exams/$examId', params: { examId } })
  }

  return (
    <AppShell title="Before You Begin" description="Review the exam details carefully before starting.">
      {isLoading && (
        <div className="flex items-center gap-2 py-10 text-xs text-gray-500">
          <Spinner /> Loading exam details...
        </div>
      )}
      {isError && (
        <Banner tone="error">{apiErrorMessage(error, 'Could not load exam.')}</Banner>
      )}

      {exam && (
        <div className="mx-auto max-w-2xl space-y-6">
          <Card className="rounded-2xl border-gray-100 shadow-sm">
            <CardHeader>
              <CardTitle className="text-xl font-bold text-gray-900">{exam.title}</CardTitle>
              <p className="text-xs text-gray-500 font-mono mt-1">
                {exam.subjectCode} · Grade {exam.gradeLevel} · {SCOPE_LABELS[exam.examScope] ?? exam.examScope}
              </p>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <div className="flex flex-col items-center gap-1 rounded-xl bg-gray-50 py-4">
                  <FileText className="h-5 w-5 text-gray-400" />
                  <span className="text-xl font-bold text-gray-900">{exam.questionCount}</span>
                  <span className="text-xs text-gray-500">Questions</span>
                </div>
                <div className="flex flex-col items-center gap-1 rounded-xl bg-gray-50 py-4">
                  <BookOpen className="h-5 w-5 text-gray-400" />
                  <span className="text-xl font-bold text-gray-900">{exam.totalMarks}</span>
                  <span className="text-xs text-gray-500">Total Marks</span>
                </div>
                <div className="flex flex-col items-center gap-1 rounded-xl bg-gray-50 py-4">
                  <Clock className="h-5 w-5 text-gray-400" />
                  <span className="text-xl font-bold text-gray-900">
                    {exam.timeLimitMinutes ? `${exam.timeLimitMinutes}m` : '–'}
                  </span>
                  <span className="text-xs text-gray-500">
                    {exam.timeLimitMinutes ? 'Time Limit' : 'No Time Limit'}
                  </span>
                </div>
              </div>

              <div className="flex items-center gap-2 text-xs text-gray-500">
                <RotateCcw className="h-3.5 w-3.5" aria-hidden />
                {exam.attemptLimit > 1
                  ? `You may attempt this exam up to ${exam.attemptLimit} times.`
                  : 'You get one attempt at this exam.'}
              </div>

              <div className="rounded-xl border border-blue-100 bg-blue-50 p-4 space-y-2">
                <div className="flex items-center gap-2 text-sm font-semibold text-blue-800">
                  <Info className="h-4 w-4 shrink-0" />
                  Instructions
                </div>
                <ul className="space-y-1.5 text-xs text-blue-700 list-disc list-inside leading-relaxed">
                  <li>Read each question carefully before answering.</li>
                  <li>For multiple-choice questions, select the best answer.</li>
                  <li>Your progress is saved automatically as you answer.</li>
                  {exam.timeLimitMinutes && (
                    <li>
                      You have {exam.timeLimitMinutes} minutes once you start -- the timer begins immediately
                      and does not pause if you close this tab.
                    </li>
                  )}
                  <li>You can return to any question before submitting.</li>
                  <li>Once submitted, you cannot change your answers.</li>
                  <li>The order of questions and answer choices is randomized for each student.</li>
                </ul>
              </div>

              <div className="rounded-xl border border-amber-100 bg-amber-50 p-3 flex items-start gap-2">
                <AlertCircle className="h-4 w-4 text-amber-600 shrink-0 mt-0.5" />
                <p className="text-xs text-amber-700">
                  After submitting, your answers will be analyzed for knowledge gaps. Your personalized
                  study plan will be updated based on your performance.
                </p>
              </div>

              <label className="flex items-start gap-2 text-xs text-gray-600 pt-1">
                <input
                  type="checkbox"
                  checked={understood}
                  onChange={(e) => setUnderstood(e.target.checked)}
                  className="mt-0.5"
                />
                <span>
                  I understand the instructions above{exam.timeLimitMinutes ? ', including the time limit,' : ''}{' '}
                  and I&apos;m ready to begin.
                </span>
              </label>
            </CardContent>
          </Card>

          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={() => void navigate({ to: '/student/exams' })}>
              Back
            </Button>
            <Button onClick={startExam} disabled={!understood} className="px-6">
              Start Exam
            </Button>
          </div>
        </div>
      )}
    </AppShell>
  )
}
