import { useQueries, useQuery } from '@tanstack/react-query'
import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { listRegions, getRegionStats } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { formatNumber } from '@lib/utils/format'

export function MinistryRegionalOverviewPage() {
  const { data: regions, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.regions(),
    queryFn: listRegions,
  })

  const regionStats = useQueries({
    queries: (regions ?? []).map((r) => ({
      queryKey: queryKeys.regionStats(r.id),
      queryFn: () => getRegionStats(r.id),
      enabled: Boolean(regions),
    })),
  })

  const totalStudents = regionStats.reduce((s, q) => s + (q.data?.student_count ?? 0), 0)
  const totalSchools = regionStats.reduce((s, q) => s + (q.data?.school_count ?? 0), 0)
  const avgMastery = regionStats.length > 0
    ? regionStats.reduce((s, q) => s + (q.data?.avg_assessment_score ?? 0), 0) / Math.max(1, regionStats.filter((q) => q.data).length)
    : 0

  return (
    <AppShell
      title="National Regional Overview & Equity Index"
      description="Cross-regional educational performance, equity indices, and curriculum adoption scorecards."
    >
      <div className="space-y-4">
        {isLoading && (
          <div className="flex items-center gap-2 py-10 text-xs text-slate-500">
            <Spinner /> Loading regional data…
          </div>
        )}
        {isError && <Banner tone="error">{(error as Error).message}</Banner>}

        {regions && (
          <>
            {/* National KPIs */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
                <p className="text-xs font-medium text-slate-500">National Student Enrollment</p>
                <p className="mt-1 text-2xl font-bold text-slate-900 font-display">
                  {totalStudents > 0 ? formatNumber(totalStudents) : '—'}
                </p>
                <p className="mt-1 text-[11px] text-slate-500">Across {regions.length} Regions</p>
              </div>
              <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
                <p className="text-xs font-medium text-slate-500">Avg Regional Mastery</p>
                <p className="mt-1 text-2xl font-bold text-teal-700 font-display">
                  {avgMastery > 0 ? `${avgMastery.toFixed(1)}%` : '—'}
                </p>
                <p className="mt-1 text-[11px] text-emerald-600 font-medium">Across all tracked regions</p>
              </div>
              <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
                <p className="text-xs font-medium text-slate-500">Total Schools</p>
                <p className="mt-1 text-2xl font-bold text-slate-900 font-display">
                  {totalSchools > 0 ? formatNumber(totalSchools) : '—'}
                </p>
                <p className="mt-1 text-[11px] text-slate-500">National directory</p>
              </div>
            </div>

            {/* Region cards */}
            {regions.length === 0 ? (
              <EmptyState title="No regions found" description="Regions will appear here once seeded." />
            ) : (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {regions.map((r, i) => {
                  const stats = regionStats[i]?.data
                  const mastery = stats?.avg_assessment_score
                  const status = mastery == null ? null
                    : mastery >= 80 ? 'Top Tier'
                    : mastery >= 70 ? 'On Track'
                    : 'Support Targeted'

                  return (
                    <div key={r.id} className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                      <div className="flex items-center justify-between">
                        <span className="font-mono text-xs font-bold text-teal-700">{r.code}</span>
                        {status && (
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${
                            status === 'Top Tier' ? 'bg-emerald-100 text-emerald-800'
                            : status === 'On Track' ? 'bg-blue-100 text-blue-800'
                            : 'bg-amber-100 text-amber-800'
                          }`}>
                            {status}
                          </span>
                        )}
                      </div>

                      <h3 className="mt-2 text-base font-bold text-slate-900">{r.name}</h3>

                      <div className="mt-3 grid grid-cols-2 gap-2 text-xs">
                        <div className="rounded-lg bg-slate-50 p-2 text-slate-600">
                          <span>Schools</span>
                          <p className="font-bold text-slate-900 text-sm mt-0.5">
                            {stats ? formatNumber(stats.school_count) : '—'}
                          </p>
                        </div>
                        <div className="rounded-lg bg-slate-50 p-2 text-slate-600">
                          <span>Avg Mastery</span>
                          <p className="font-bold text-teal-700 text-sm mt-0.5">
                            {stats ? `${(stats.avg_assessment_score ?? 0).toFixed(1)}%` : '—'}
                          </p>
                        </div>
                        <div className="rounded-lg bg-slate-50 p-2 text-slate-600">
                          <span>Students</span>
                          <p className="font-bold text-slate-900 text-sm mt-0.5">
                            {stats ? formatNumber(stats.student_count) : '—'}
                          </p>
                        </div>
                        <div className="rounded-lg bg-slate-50 p-2 text-slate-600">
                          <span>Teachers</span>
                          <p className="font-bold text-slate-900 text-sm mt-0.5">
                            {stats ? formatNumber(stats.teacher_count) : '—'}
                          </p>
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </>
        )}
      </div>
    </AppShell>
  )
}
