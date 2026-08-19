import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { BookOpen, ExternalLink } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { listSubjects } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function MinistryCurriculumGovernancePage() {
  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.subjects(),
    queryFn: listSubjects,
  })

  const subjects = data ?? []
  const activeCount = subjects.filter((s) => s.isCurrent).length
  const totalClos = subjects.reduce((sum, s) => sum + s.cloCount, 0)

  return (
    <AppShell
      title="National Curriculum Standards & Governance"
      description="Cross-regional deployment tracking of curriculum versions, standards alignment, and CLO coverage."
    >
      <div className="space-y-6">
        {isError && (
          <Banner
            variant="error"
            title="Failed to load curriculum subjects"
            description="Try refreshing the page."
          />
        )}

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <p className="text-xs font-medium text-slate-500">Active National Subjects</p>
            <p className="mt-1 text-2xl font-bold text-slate-900">{activeCount}</p>
            <p className="text-xs text-slate-400">Currently deployed</p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <p className="text-xs font-medium text-slate-500">Total CLO Standards</p>
            <p className="mt-1 text-2xl font-bold text-teal-700">{totalClos.toLocaleString()}</p>
            <p className="text-xs text-slate-400">Across all active subjects</p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <p className="text-xs font-medium text-slate-500">Total Subjects</p>
            <p className="mt-1 text-2xl font-bold text-slate-900">{subjects.length}</p>
            <p className="text-xs text-slate-400">All versions</p>
          </div>
        </div>

        <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
          {isLoading ? (
            <div className="flex justify-center py-12">
              <Spinner />
            </div>
          ) : subjects.length === 0 ? (
            <EmptyState
              icon={BookOpen}
              title="No curriculum subjects"
              description="Approved curriculum subjects will appear here."
            />
          ) : (
            <table className="w-full text-left text-xs text-slate-600">
              <thead className="border-b border-slate-100 bg-slate-50 text-[11px] font-bold uppercase tracking-wider text-slate-500">
                <tr>
                  <th className="px-4 py-3">Subject & Code</th>
                  <th className="px-4 py-3">Grade</th>
                  <th className="px-4 py-3 text-center">Version</th>
                  <th className="px-4 py-3 text-center">CLOs</th>
                  <th className="px-4 py-3 text-center">Status</th>
                  <th className="px-4 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 font-normal">
                {subjects.map((s) => (
                  <tr key={s.code} className="hover:bg-slate-50/80">
                    <td className="px-4 py-3">
                      <span className="font-mono text-[11px] font-bold text-teal-700 block">{s.code}</span>
                      <span className="font-semibold text-slate-900">{s.nameEn}</span>
                    </td>
                    <td className="px-4 py-3">Grade {s.gradeLevel}</td>
                    <td className="px-4 py-3 text-center font-mono font-bold text-slate-700">v{s.version}</td>
                    <td className="px-4 py-3 text-center font-mono">{s.cloCount}</td>
                    <td className="px-4 py-3 text-center">
                      <span
                        className={`rounded-full px-2.5 py-0.5 text-[10px] font-bold ${
                          s.isCurrent
                            ? 'bg-emerald-100 text-emerald-800'
                            : 'bg-slate-100 text-slate-500'
                        }`}
                      >
                        {s.isCurrent ? 'Active' : 'Superseded'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Link
                        to="/curriculum/versions"
                        search={{ code: s.code }}
                        className="inline-flex items-center gap-1 text-teal-700 font-bold hover:underline text-xs"
                      >
                        View Versions <ExternalLink className="h-3 w-3" />
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </AppShell>
  )
}
