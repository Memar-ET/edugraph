import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Users } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { listAllStudents } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'

const GRADES = [7, 8, 9, 10, 11, 12]

export function SchoolClassesPage() {
  const user = useAuthStore((s) => s.user)
  const schoolId = user?.school_id ?? ''
  const [gradeFilter, setGradeFilter] = useState<number | undefined>(undefined)

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.allStudents({ school_id: schoolId, grade_level: gradeFilter }),
    queryFn: () => listAllStudents({ school_id: schoolId, grade_level: gradeFilter, limit: 200 }),
    enabled: Boolean(schoolId),
    select: (r) => r.items,
  })

  const students = data ?? []

  const byGrade = GRADES.map((g) => ({
    grade: g,
    count: students.filter((s) => s.grade_level === g).length,
  }))

  return (
    <AppShell title="Classes & Sections" description="Student enrollment breakdown by grade level for your school.">
      <div className="space-y-5">
        {/* Grade filter */}
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => setGradeFilter(undefined)}
            className={`rounded-xl px-3 py-1.5 text-xs font-semibold transition-all ${!gradeFilter ? 'bg-teal-700 text-white' : 'border border-slate-200 bg-white text-slate-700 hover:bg-slate-50'}`}
          >
            All Grades
          </button>
          {GRADES.map((g) => (
            <button
              key={g}
              type="button"
              onClick={() => setGradeFilter(g)}
              className={`rounded-xl px-3 py-1.5 text-xs font-semibold transition-all ${gradeFilter === g ? 'bg-teal-700 text-white' : 'border border-slate-200 bg-white text-slate-700 hover:bg-slate-50'}`}
            >
              Grade {g}
            </button>
          ))}
        </div>

        {/* Grade summary cards */}
        {!gradeFilter && (
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
            {byGrade.map(({ grade, count }) => (
              <button
                key={grade}
                type="button"
                onClick={() => setGradeFilter(grade)}
                className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm text-left hover:border-teal-400 transition-all"
              >
                <p className="text-xs font-medium text-slate-500">Grade {grade}</p>
                <p className="mt-1 text-2xl font-bold font-display text-slate-900">{count}</p>
                <p className="mt-0.5 text-[11px] text-slate-400">Students</p>
              </button>
            ))}
          </div>
        )}

        {isLoading && (
          <div className="flex items-center gap-2 py-10 text-xs text-slate-500">
            <Spinner /> Loading students...
          </div>
        )}
        {isError && <Banner tone="error">Could not load students. Please try again.</Banner>}
        {!isLoading && students.length === 0 && (
          <EmptyState
            icon={Users}
            title="No students found"
            description={gradeFilter ? `No students enrolled in Grade ${gradeFilter}.` : 'No students enrolled yet.'}
          />
        )}

        {students.length > 0 && (
          <div className="rounded-2xl border border-slate-200 bg-white shadow-sm overflow-hidden">
            <div className="px-5 py-4 border-b border-slate-100 flex items-center justify-between">
              <h3 className="font-bold text-sm text-slate-900">
                {gradeFilter ? `Grade ${gradeFilter} Students` : 'All Students'}
              </h3>
              <span className="text-xs text-slate-500">{students.length} students</span>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-xs text-left">
                <thead className="border-b border-slate-100 bg-slate-50 text-[10px] uppercase font-bold text-slate-400">
                  <tr>
                    <th className="px-4 py-3">Admission No.</th>
                    <th className="px-4 py-3">Name</th>
                    <th className="px-4 py-3">Grade</th>
                    <th className="px-4 py-3">Email</th>
                    <th className="px-4 py-3">Enrolled</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {students.map((s) => (
                    <tr key={s.id} className="hover:bg-slate-50">
                      <td className="px-4 py-3 font-mono font-bold text-teal-700">{s.admission_no}</td>
                      <td className="px-4 py-3 font-medium text-slate-900">{s.full_name ?? '—'}</td>
                      <td className="px-4 py-3 text-slate-600">Grade {s.grade_level}</td>
                      <td className="px-4 py-3 text-slate-400">{s.email ?? '—'}</td>
                      <td className="px-4 py-3 text-slate-400 font-mono">{new Date(s.created_at).toLocaleDateString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </AppShell>
  )
}
