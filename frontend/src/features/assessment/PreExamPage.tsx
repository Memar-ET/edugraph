import { useNavigate, useParams } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { AlertCircle, BookOpen, Clock, FileText, Info } from 'lucide-react'

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

  const { data: exam, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.exam(examId),
    queryFn: () => getExam(examId),
  })

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
                  <span className="text-xl font-bold text-gray-900">–</span>
                  <span className="text-xs text-gray-500">No Timer</span>
                </div>
              </div>

              <div className="rounded-xl border border-blue-100 bg-blue-50 p-4 space-y-2">
                <div className="flex items-center gap-2 text-sm font-semibold text-blue-800">
                  <Info className="h-4 w-4 shrink-0" />
                  Instructions
                </div>
                <ul className="space-y-1.5 text-xs text-blue-700 list-disc list-inside leading-relaxed">
                  <li>Read each question carefully before answering.</li>
                  <li>For multiple-choice questions, select the best answer.</li>
                  <li>Your progress is saved automatically every 30 seconds.</li>
                  <li>You can return to any question before submitting.</li>
                  <li>Once submitted, you cannot change your answers.</li>
                </ul>
              </div>

              <div className="rounded-xl border border-amber-100 bg-amber-50 p-3 flex items-start gap-2">
                <AlertCircle className="h-4 w-4 text-amber-600 shrink-0 mt-0.5" />
                <p className="text-xs text-amber-700">
                  After submitting, your answers will be analyzed for knowledge gaps. Your personalized
                  study plan will be updated based on your performance.
                </p>
              </div>
            </CardContent>
          </Card>

          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={() => void navigate({ to: '/student/exams' })}>
              Back
            </Button>
            <Button onClick={startExam} className="px-6">
              Start Exam
            </Button>
          </div>
        </div>
      )}
    </AppShell>
  )
}
