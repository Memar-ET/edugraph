import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Plus, Search, Printer, X, Eye, PenLine, BarChart2 } from 'lucide-react'
import { AppShell } from '@components/layout'
import { KpiTrendCard } from '@components/dashboard/KpiTrendCard'
import { ExamWizard } from '@components/shared/ExamWizard'
import { listExams, closeExam, fetchExamPrintBlobUrl } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function TeacherExamListPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [wizardOpen, setWizardOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [page] = useState(1)

  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.exams(),
    queryFn: () => listExams(page, 50),
  })

  const closeMutation = useMutation({
    mutationFn: (examId: string) => closeExam(examId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.exams() }),
  })

  const printMutation = useMutation({
    mutationFn: (examId: string) => fetchExamPrintBlobUrl(examId),
    onSuccess: (url) => {
      const win = window.open(url, '_blank')
      if (win) win.focus()
    },
  })

  const items = (data?.items ?? []).filter((exam) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      exam.title.toLowerCase().includes(q) ||
      exam.subjectCode.toLowerCase().includes(q) ||
      String(exam.gradeLevel).includes(q)
    )
  })

  const published = items.filter((e) => e.status === 'published').length
  const drafts = items.filter((e) => e.status === 'draft').length
  const closed = items.filter((e) => e.status === 'closed').length
  const totalSubmissions = items.reduce((sum, e) => sum + (e.submissionCount ?? 0), 0)

  return (
    <AppShell
      actions={
        <button
          type="button"
          onClick={() => setWizardOpen(true)}
          className="flex items-center gap-1.5 rounded-xl bg-teal-700 px-3.5 py-1.5 text-xs font-semibold text-white hover:bg-teal-800"
        >
          <Plus className="h-4 w-4" /> Create Exam
        </button>
      }
    >
      <ExamWizard
        isOpen={wizardOpen}
        onClose={() => {
          setWizardOpen(false)
          queryClient.invalidateQueries({ queryKey: queryKeys.exams() })
        }}
      />

      <div className="space-y-6">
        <div>
          <h1 className="font-display font-extrabold text-2xl text-slate-900 tracking-tight">
            Exam Authoring &amp; Management
          </h1>
          <p className="text-xs text-slate-500 mt-1">
            Author assessment items, launch exam creation wizard, and manage publication schedules.
          </p>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-4">
          <KpiTrendCard
            title="Published Exams"
            value={String(published)}
            subtitle={`${closed} closed`}
            sparklineData={[0, 0, published]}
          />
          <KpiTrendCard
            title="Draft Exams"
            value={String(drafts)}
            subtitle="Awaiting publish"
            sparklineData={[0, 0, drafts]}
          />
          <KpiTrendCard
            title="Total Submissions"
            value={String(totalSubmissions)}
            subtitle="Across all exams"
            sparklineData={[0, 0, totalSubmissions]}
          />
          <KpiTrendCard
            title="Total Exams"
            value={String(data?.items.length ?? 0)}
            subtitle="All statuses"
            sparklineData={[0, 0, data?.items.length ?? 0]}
          />
        </div>

        <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <h3 className="font-bold text-sm text-slate-900">All Assessments</h3>
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search exams…"
                className="rounded-xl border border-slate-200 bg-slate-50 pl-8 pr-3 py-1.5 text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-teal-500"
              />
            </div>
          </div>

          {isError && (
            <div className="rounded-2xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800">
              Failed to load exams: {(error as Error).message}
            </div>
          )}

          {isLoading ? (
            <div className="space-y-2">
              {[...Array(4)].map((_, i) => (
                <div key={i} className="h-12 animate-pulse rounded-xl bg-slate-100" />
              ))}
            </div>
          ) : items.length === 0 ? (
            <div className="py-12 text-center text-xs text-slate-500">
              {search ? 'No exams match your search.' : 'No exams yet. Click "Create Exam" to upload one.'}
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="border-b border-slate-100 bg-slate-50 text-[10px] uppercase font-bold text-slate-400">
                  <tr>
                    <th className="p-3">Title</th>
                    <th className="p-3">Subject / Grade</th>
                    <th className="p-3">Questions</th>
                    <th className="p-3">Submissions</th>
                    <th className="p-3">Status</th>
                    <th className="p-3">Created</th>
                    <th className="p-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 font-medium text-slate-700">
                  {items.map((exam) => (
                    <tr key={exam.examId} className="hover:bg-slate-50">
                      <td className="p-3 font-bold text-slate-900 max-w-[200px] truncate">
                        <span className="font-mono text-teal-700 block text-[10px]">{exam.examId.slice(0, 8)}…</span>
                        {exam.title}
                      </td>
                      <td className="p-3">{exam.subjectCode} (G{exam.gradeLevel})</td>
                      <td className="p-3">{exam.questionCount}</td>
                      <td className="p-3">{exam.submissionCount ?? 0}</td>
                      <td className="p-3">
                        <span
                          className={`rounded-full px-2.5 py-0.5 text-[10px] font-bold uppercase ${
                            exam.status === 'published'
                              ? 'bg-emerald-100 text-emerald-800'
                              : exam.status === 'draft'
                              ? 'bg-amber-100 text-amber-800'
                              : exam.status === 'closed'
                              ? 'bg-slate-100 text-slate-600'
                              : 'bg-slate-100 text-slate-500'
                          }`}
                        >
                          {exam.status}
                        </span>
                      </td>
                      <td className="p-3 font-mono text-slate-400">
                        {new Date(exam.createdAt).toLocaleDateString()}
                      </td>
                      <td className="p-3">
                        <div className="flex items-center justify-end gap-1">
                          <button
                            type="button"
                            title="Review exam"
                            onClick={() =>
                              navigate({ to: '/teacher/exams/$examId', params: { examId: exam.examId } })
                            }
                            className="rounded-lg p-1.5 text-slate-400 hover:text-teal-700 hover:bg-teal-50"
                          >
                            <Eye className="h-3.5 w-3.5" />
                          </button>
                          {(exam.status === 'draft') && (
                            <button
                              type="button"
                              title="Grade exam"
                              onClick={() =>
                                navigate({ to: '/teacher/exams/$examId/grade', params: { examId: exam.examId } })
                              }
                              className="rounded-lg p-1.5 text-slate-400 hover:text-teal-700 hover:bg-teal-50"
                            >
                              <PenLine className="h-3.5 w-3.5" />
                            </button>
                          )}
                          {(exam.status === 'published' || exam.status === 'closed') && (
                            <>
                              <button
                                type="button"
                                title="Exam quality"
                                onClick={() =>
                                  navigate({ to: '/teacher/exams/$examId/quality', params: { examId: exam.examId } })
                                }
                                className="rounded-lg p-1.5 text-slate-400 hover:text-teal-700 hover:bg-teal-50"
                              >
                                <BarChart2 className="h-3.5 w-3.5" />
                              </button>
                              <button
                                type="button"
                                title="Print exam sheet"
                                disabled={printMutation.isPending}
                                onClick={() => printMutation.mutate(exam.examId)}
                                className="rounded-lg p-1.5 text-slate-400 hover:text-teal-700 hover:bg-teal-50 disabled:opacity-50"
                              >
                                <Printer className="h-3.5 w-3.5" />
                              </button>
                            </>
                          )}
                          {exam.status === 'published' && (
                            <button
                              type="button"
                              title="Close exam"
                              disabled={closeMutation.isPending}
                              onClick={() => {
                                if (confirm('Close this exam? Students will no longer be able to submit.')) {
                                  closeMutation.mutate(exam.examId)
                                }
                              }}
                              className="rounded-lg p-1.5 text-slate-400 hover:text-rose-700 hover:bg-rose-50 disabled:opacity-50"
                            >
                              <X className="h-3.5 w-3.5" />
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </AppShell>
  )
}
