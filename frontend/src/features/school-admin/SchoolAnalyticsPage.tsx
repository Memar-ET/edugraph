import { useQuery } from '@tanstack/react-query'
import { BarChart2 } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { getSchoolQualityScores } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'

export function SchoolAnalyticsPage() {
  const user = useAuthStore((s) => s.user)
  const schoolId = user?.school_id ?? ''

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.schoolQuality(schoolId),
    queryFn: () => getSchoolQualityScores(schoolId),
    enabled: Boolean(schoolId),
  })

  return (
    <AppShell title="School Analytics" description="Quality scores and curriculum alignment breakdown by subject and grade.">
      <div className="space-y-5">
        {isLoading && (
          <div className="flex items-center gap-2 py-10 text-xs text-slate-500">
            <Spinner /> Loading analytics...
          </div>
        )}
        {isError && <Banner tone="error">Could not load school analytics. Please try again.</Banner>}
        {data && data.scores.length === 0 && (
          <EmptyState
            icon={BarChart2}
            title="No analytics data yet"
            description="Quality scores compute automatically once exams have been published and graded."
          />
        )}
        {data && data.scores.length > 0 && (
          <>
            {/* Summary row */}
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
              {[
                { label: 'Avg Composite', value: (data.scores.reduce((s, q) => s + (q.compositeScore ?? 0), 0) / data.scores.length * 100).toFixed(1) + '%', color: 'text-teal-700' },
                { label: 'Avg CLO Coverage', value: (data.scores.reduce((s, q) => s + (q.cloCoveragePct ?? 0), 0) / data.scores.length * 100).toFixed(1) + '%', color: 'text-emerald-700' },
                { label: 'Avg Student Mastery', value: (data.scores.reduce((s, q) => s + (q.studentMasteryPct ?? 0), 0) / data.scores.length * 100).toFixed(1) + '%', color: 'text-blue-700' },
                { label: 'Avg Exam Quality', value: (data.scores.reduce((s, q) => s + (q.examQualityAvg ?? 0), 0) / data.scores.length * 100).toFixed(1) + '%', color: 'text-violet-700' },
              ].map(({ label, value, color }) => (
                <div key={label} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
                  <p className="text-xs font-medium text-slate-500">{label}</p>
                  <p className={`mt-1 text-3xl font-bold font-display ${color}`}>{value}</p>
                </div>
              ))}
            </div>

            {/* Per-subject breakdown */}
            <div className="rounded-2xl border border-slate-200 bg-white shadow-sm overflow-hidden">
              <div className="px-5 py-4 border-b border-slate-100">
                <h3 className="font-bold text-sm text-slate-900">Subject & Grade Breakdown</h3>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-xs text-left">
                  <thead className="border-b border-slate-100 bg-slate-50 text-[10px] uppercase font-bold text-slate-400">
                    <tr>
                      <th className="px-4 py-3">Subject</th>
                      <th className="px-4 py-3">Grade</th>
                      <th className="px-4 py-3 text-center">Composite</th>
                      <th className="px-4 py-3 text-center">CLO Coverage</th>
                      <th className="px-4 py-3 text-center">Student Mastery</th>
                      <th className="px-4 py-3 text-center">Exam Quality</th>
                      <th className="px-4 py-3 text-center">Compliance</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {data.scores.map((s, i) => (
                      <tr key={i} className="hover:bg-slate-50">
                        <td className="px-4 py-3 font-semibold text-slate-900">{s.subjectCode ?? '—'}</td>
                        <td className="px-4 py-3 text-slate-500">{s.gradeLevel ? `Grade ${s.gradeLevel}` : '—'}</td>
                        <td className="px-4 py-3 text-center font-bold text-teal-700">{((s.compositeScore ?? 0) * 100).toFixed(1)}%</td>
                        <td className="px-4 py-3 text-center text-emerald-700 font-semibold">{((s.cloCoveragePct ?? 0) * 100).toFixed(1)}%</td>
                        <td className="px-4 py-3 text-center text-blue-700 font-semibold">{((s.studentMasteryPct ?? 0) * 100).toFixed(1)}%</td>
                        <td className="px-4 py-3 text-center text-violet-700 font-semibold">{((s.examQualityAvg ?? 0) * 100).toFixed(1)}%</td>
                        <td className="px-4 py-3 text-center">
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${s.flaggedForReview ? 'bg-rose-100 text-rose-800' : 'bg-emerald-100 text-emerald-800'}`}>
                            {s.flaggedForReview ? 'Flagged' : 'OK'}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </>
        )}
      </div>
    </AppShell>
  )
}
