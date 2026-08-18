import { ClipboardList, Play, CheckCircle2 } from 'lucide-react'
import { useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { AppShell } from '@components/layout'
import { KpiTrendCard } from '@components/dashboard/KpiTrendCard'
import { getAvailableExams } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function StudentExamCenterPage() {
  const navigate = useNavigate()
  const { data, isLoading, error } = useQuery({
    queryKey: queryKeys.availableExams(),
    queryFn: getAvailableExams,
  })

  const exams = data?.exams ?? []
  const available = exams.filter((e) => !e.alreadyAttempted)
  const attempted = exams.filter((e) => e.alreadyAttempted)

  if (isLoading) {
    return (
      <AppShell>
        <div className="space-y-4 animate-pulse">
          <div className="h-8 bg-slate-200 rounded w-64" />
          <div className="grid grid-cols-3 gap-4">
            {[0, 1, 2].map((i) => <div key={i} className="h-24 bg-slate-200 rounded-2xl" />)}
          </div>
          <div className="h-64 bg-slate-200 rounded-2xl" />
        </div>
      </AppShell>
    )
  }

  if (error) {
    return (
      <AppShell>
        <div className="rounded-2xl border border-rose-200 bg-rose-50 p-5 text-rose-800 text-sm">
          Failed to load exams: {(error as Error).message}
        </div>
      </AppShell>
    )
  }

  return (
    <AppShell>
      <div className="space-y-6">
        <div>
          <h1 className="font-display font-extrabold text-2xl text-slate-900 tracking-tight">Student Exam Center</h1>
          <p className="text-xs text-slate-500 mt-1">Your available assessments and exam history.</p>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <KpiTrendCard
            title="Available Exams"
            value={`${available.length} Active`}
            subtitle="Ready for you to take"
            sparklineData={[available.length]}
          />
          <KpiTrendCard
            title="Completed Exams"
            value={`${attempted.length} Done`}
            subtitle="Already submitted"
            sparklineData={[attempted.length]}
          />
          <KpiTrendCard
            title="Total Assessments"
            value={`${exams.length} Total`}
            subtitle="Across all subjects"
            sparklineData={[exams.length]}
          />
        </div>

        {exams.length === 0 ? (
          <div className="rounded-2xl border border-slate-200 bg-white p-10 text-center shadow-sm">
            <ClipboardList className="mx-auto h-10 w-10 text-slate-300 mb-3" />
            <p className="font-semibold text-slate-700">No exams available</p>
            <p className="text-xs text-slate-500 mt-1">Your teacher hasn't published any exams yet.</p>
          </div>
        ) : (
          <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm space-y-4">
            <h3 className="font-bold text-sm text-slate-900">Available & Scheduled Exams</h3>
            <div className="divide-y divide-slate-100">
              {exams.map((exam) => (
                <div key={exam.examId} className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 py-4">
                  <div className="flex items-start gap-3">
                    <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl font-bold text-xs ${
                      exam.alreadyAttempted ? 'bg-slate-100 text-slate-700' : 'bg-emerald-100 text-emerald-800'
                    }`}>
                      <ClipboardList className="h-5 w-5" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-mono text-[11px] font-bold text-slate-400">{exam.subjectCode}</span>
                        <span className="font-bold text-xs text-slate-900">{exam.title}</span>
                      </div>
                      <p className="text-xs text-slate-500 mt-0.5">
                        Grade {exam.gradeLevel} • {exam.questionCount} Questions • {exam.totalMarks} Marks
                      </p>
                      {exam.closesAt && (
                        <p className="text-[11px] font-medium text-amber-600 mt-0.5">
                          Closes {new Date(exam.closesAt).toLocaleDateString()}
                        </p>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center gap-3">
                    {exam.alreadyAttempted ? (
                      <span className="flex items-center gap-1.5 rounded-full bg-emerald-50 px-3 py-1 text-xs font-semibold text-emerald-700 border border-emerald-200">
                        <CheckCircle2 className="h-3.5 w-3.5" /> Submitted
                      </span>
                    ) : (
                      <button
                        type="button"
                        onClick={() => void navigate({ to: '/student/exams/$examId/pre', params: { examId: exam.examId } })}
                        className="flex items-center gap-1.5 rounded-xl bg-teal-700 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-teal-800"
                      >
                        <Play className="h-3.5 w-3.5 fill-current" /> Start Exam
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </AppShell>
  )
}
