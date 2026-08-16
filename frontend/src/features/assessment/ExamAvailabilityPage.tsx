import { useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { BookOpen, CheckCircle2, Clock, FileText } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Button, Card, CardContent, EmptyState, Spinner } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { getAvailableExams } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import type { ExamAvailability } from '@/types/api'

const SCOPE_LABELS: Record<string, string> = {
  unit_test: 'Unit Test',
  midterm: 'Midterm',
  final_exam: 'Final Exam',
}

function ExamCard({ exam }: { exam: ExamAvailability }) {
  const navigate = useNavigate()

  const goToPreExam = () => {
    void navigate({ to: '/student/exams/$examId/pre', params: { examId: exam.examId } })
  }

  return (
    <Card className="rounded-2xl border-gray-100 shadow-sm">
      <CardContent className="p-5">
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <span className="inline-flex rounded-full bg-indigo-50 px-2 py-0.5 text-[11px] font-semibold text-indigo-700">
                {SCOPE_LABELS[exam.examScope] ?? exam.examScope}
              </span>
              <span className="text-[11px] text-gray-400 font-mono">
                {exam.subjectCode} · Grade {exam.gradeLevel}
              </span>
            </div>
            <h3 className="font-semibold text-gray-900 truncate">{exam.title}</h3>
            <div className="mt-2 flex items-center gap-4 text-xs text-gray-500">
              <span className="flex items-center gap-1">
                <FileText className="h-3.5 w-3.5" />
                {exam.questionCount} questions
              </span>
              <span className="flex items-center gap-1">
                <BookOpen className="h-3.5 w-3.5" />
                {exam.totalMarks} marks
              </span>
              {exam.closesAt && (
                <span className="flex items-center gap-1 text-amber-600">
                  <Clock className="h-3.5 w-3.5" />
                  Closes {new Date(exam.closesAt).toLocaleDateString()}
                </span>
              )}
            </div>
          </div>

          <div className="shrink-0">
            {exam.alreadyAttempted ? (
              <span className="flex items-center gap-1.5 rounded-full bg-emerald-50 px-3 py-1.5 text-xs font-semibold text-emerald-700">
                <CheckCircle2 className="h-3.5 w-3.5" />
                Submitted
              </span>
            ) : (
              <Button size="sm" onClick={goToPreExam}>
                Start Exam
              </Button>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

export function ExamAvailabilityPage() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.availableExams(),
    queryFn: getAvailableExams,
    staleTime: 60_000,
  })

  const pending = data?.exams.filter((e) => !e.alreadyAttempted) ?? []
  const completed = data?.exams.filter((e) => e.alreadyAttempted) ?? []

  return (
    <AppShell
      title="Available Exams"
      description="Published exams you can take. Click an exam to review the details before starting."
    >
      {isLoading && (
        <div className="flex items-center gap-2 py-10 text-xs text-gray-500">
          <Spinner /> Loading available exams...
        </div>
      )}
      {isError && (
        <Banner tone="error">{apiErrorMessage(error, 'Could not load available exams.')}</Banner>
      )}

      {data && (
        <div className="space-y-8">
          <section>
            <h2 className="mb-3 text-sm font-bold text-gray-700 uppercase tracking-wide">
              Pending ({pending.length})
            </h2>
            {pending.length === 0 ? (
              <EmptyState title="No pending exams" description="All exams have been submitted." />
            ) : (
              <div className="space-y-3">
                {pending.map((exam) => (
                  <ExamCard key={exam.examId} exam={exam} />
                ))}
              </div>
            )}
          </section>

          {completed.length > 0 && (
            <section>
              <h2 className="mb-3 text-sm font-bold text-gray-700 uppercase tracking-wide">
                Completed ({completed.length})
              </h2>
              <div className="space-y-3">
                {completed.map((exam) => (
                  <ExamCard key={exam.examId} exam={exam} />
                ))}
              </div>
            </section>
          )}
        </div>
      )}
    </AppShell>
  )
}
