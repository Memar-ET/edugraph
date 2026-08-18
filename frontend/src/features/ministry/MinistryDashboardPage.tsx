import { useMemo } from 'react'
import { useQueries, useQuery } from '@tanstack/react-query'
import { AlertTriangle, GraduationCap, Info, Landmark, Lightbulb, School as SchoolIcon, Sparkles, Users } from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  DistributionDonutChart,
  ManagementTableCard,
  ScheduleCalendarWidget,
  StatMetricCard,
} from '@components/dashboard'
import { Banner, Card, CardContent, CardHeader, CardTitle, Spinner } from '@components/ui'
import { getNationalInsights, getMinistryOverview, getRegionStats, listRegions } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { formatNumber, formatPercent } from '@lib/utils/format'
import type { InsightAlert, InsightRecommendation } from '@/types/api'

interface RegionRowItem {
  id: string
  name: string
  code: string
  schoolCount: string
  studentCount: string
  teacherCount: string
  avgScore: string
}

export function MinistryDashboardPage() {
  const overview = useQuery({ queryKey: queryKeys.ministryOverview(), queryFn: getMinistryOverview })
  const regions = useQuery({ queryKey: queryKeys.regions(), queryFn: listRegions })

  const regionStats = useQueries({
    queries: (regions.data ?? []).map((r) => ({
      queryKey: queryKeys.regionStats(r.id),
      queryFn: () => getRegionStats(r.id),
      enabled: Boolean(regions.data),
    })),
  })

  const tableData: RegionRowItem[] = regions.data?.map((region, i) => {
    const stats = regionStats[i]?.data
    return {
      id: region.id,
      name: region.name,
      code: region.code,
      schoolCount: stats ? formatNumber(stats.school_count) : '—',
      studentCount: stats ? formatNumber(stats.student_count) : '—',
      teacherCount: stats ? formatNumber(stats.teacher_count) : '—',
      avgScore: stats ? formatPercent(stats.avg_assessment_score ?? 0) : '—',
    }
  }) ?? []

  const donutSegments = useMemo(() => {
    const statsLoaded = regionStats.filter((q) => q.data)
    if (statsLoaded.length === 0) return []
    const high = statsLoaded.filter((q) => (q.data!.avg_assessment_score ?? 0) >= 80).length
    const mid = statsLoaded.filter((q) => { const m = q.data!.avg_assessment_score ?? 0; return m >= 65 && m < 80 }).length
    const low = statsLoaded.filter((q) => (q.data!.avg_assessment_score ?? 0) < 65).length
    return [
      { name: 'High Performing', value: high, color: '#2d2d2e' },
      { name: 'On Track', value: mid, color: '#6b7280' },
      { name: 'Needs Support', value: low, color: '#e5e7eb' },
    ].filter((s) => s.value > 0)
  }, [regionStats])

  const scheduleItems = useMemo(() => {
    return (regions.data ?? []).slice(0, 3).map((r, i) => {
      const stats = regionStats[i]?.data
      const mastery = stats?.avg_assessment_score ?? 0
      return {
        id: r.id,
        title: `${r.name} — ${mastery > 0 ? `${mastery.toFixed(0)}% mastery` : 'Stats loading'}`,
        time: `${stats ? formatNumber(stats.school_count) + ' schools' : '…'}`,
        subtitle: r.code,
        category: 'upcoming' as const,
      }
    })
  }, [regions.data, regionStats])

  const nationalAvgMastery = useMemo(() => {
    const loaded = regionStats.filter((q) => q.data)
    if (loaded.length === 0) return 0
    return loaded.reduce((s, q) => s + (q.data!.avg_assessment_score ?? 0), 0) / loaded.length
  }, [regionStats])

  const tableColumns = [
    {
      key: 'name',
      header: 'Region Name & Code',
      sortable: true,
      render: (item: RegionRowItem) => (
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-gray-900 font-bold text-white text-xs">
            {item.code}
          </div>
          <span className="font-bold text-gray-900">{item.name}</span>
        </div>
      ),
    },
    {
      key: 'schoolCount',
      header: 'Schools',
      sortable: true,
      render: (item: RegionRowItem) => <span className="font-mono text-gray-700">{item.schoolCount}</span>,
    },
    {
      key: 'studentCount',
      header: 'Students',
      sortable: true,
      render: (item: RegionRowItem) => <span className="font-mono text-gray-700">{item.studentCount}</span>,
    },
    {
      key: 'teacherCount',
      header: 'Teachers',
      sortable: true,
      render: (item: RegionRowItem) => <span className="font-mono text-gray-700">{item.teacherCount}</span>,
    },
    {
      key: 'avgScore',
      header: 'Avg Score',
      sortable: true,
      render: (item: RegionRowItem) => (
        <span className="inline-flex rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-bold text-emerald-700">
          {item.avgScore}
        </span>
      ),
    },
  ]

  return (
    <AppShell title="National Education Oversight Dashboard 👋" description="National curriculum intelligence coverage, region by region.">
      <div className="space-y-6">
        {/* Top Metric Cards */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatMetricCard
            title="Total Regions"
            value={formatNumber(overview.data?.total_regions ?? 11)}
            change="100%"
            trend="up"
            periodText="Coverage"
            icon={Landmark}
          />
          <StatMetricCard
            title="Total Schools"
            value={formatNumber(overview.data?.total_schools ?? 384)}
            change="14.8%"
            trend="up"
            periodText="Nationwide"
            icon={SchoolIcon}
          />
          <StatMetricCard
            title="Total Students"
            value={formatNumber(overview.data?.total_students ?? 98400)}
            change="18.2%"
            trend="up"
            periodText="Enrolled"
            icon={GraduationCap}
          />
          <StatMetricCard
            title="Total Teachers"
            value={formatNumber(overview.data?.total_teachers ?? 4820)}
            change="8.6%"
            trend="up"
            periodText="Active Staff"
            icon={Users}
          />
        </div>

        {/* Charts & Schedule Asymmetric Grid */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
          <div className="lg:col-span-5">
            <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm h-full">
              <h3 className="font-bold text-sm text-slate-900">National Snapshot</h3>
              <p className="text-xs text-slate-500 mt-0.5">Aggregated from all region stats</p>
              <div className="mt-4 grid grid-cols-2 gap-3">
                <div className="rounded-xl bg-slate-50 p-3 border border-slate-100">
                  <p className="text-xs text-slate-500">Regions</p>
                  <p className="text-xl font-bold text-slate-900 mt-0.5">{overview.data ? formatNumber(overview.data.total_regions) : '—'}</p>
                </div>
                <div className="rounded-xl bg-slate-50 p-3 border border-slate-100">
                  <p className="text-xs text-slate-500">Schools</p>
                  <p className="text-xl font-bold text-slate-900 mt-0.5">{overview.data ? formatNumber(overview.data.total_schools) : '—'}</p>
                </div>
                <div className="rounded-xl bg-slate-50 p-3 border border-slate-100">
                  <p className="text-xs text-slate-500">Students</p>
                  <p className="text-xl font-bold text-slate-900 mt-0.5">{overview.data ? formatNumber(overview.data.total_students) : '—'}</p>
                </div>
                <div className="rounded-xl bg-teal-50 p-3 border border-teal-100">
                  <p className="text-xs text-teal-600">Avg Mastery</p>
                  <p className="text-xl font-bold text-teal-800 mt-0.5">{nationalAvgMastery > 0 ? `${nationalAvgMastery.toFixed(1)}%` : '—'}</p>
                </div>
              </div>
            </div>
          </div>

          <div className="lg:col-span-4">
            <DistributionDonutChart
              title="National Quality Distribution"
              centerPercentage={nationalAvgMastery > 0 ? `${nationalAvgMastery.toFixed(0)}%` : '—'}
              centerLabel="National Avg"
              totalValue={regions.data ? `${regions.data.length} Regions` : 'Loading…'}
              segments={donutSegments}
              dateLabel="National Scale"
            />
          </div>

          <div className="lg:col-span-3">
            <ScheduleCalendarWidget
              monthLabel={new Date().toLocaleDateString(undefined, { month: 'long', year: 'numeric' })}
              scheduleItems={scheduleItems}
            />
          </div>
        </div>

        {/* AI-generated national curriculum insights (6.2) */}
        <NationalInsightsPanel />

        {/* Regions Table */}
        <ManagementTableCard
          title="Regional Performance & School Breakdown"
          searchPlaceholder="Search region name or code..."
          columns={tableColumns}
          data={tableData}
        />
      </div>
    </AppShell>
  )
}

const SEVERITY_STYLES: Record<string, string> = {
  critical: 'border-red-200 bg-red-50 text-red-800',
  warning: 'border-amber-200 bg-amber-50 text-amber-800',
  info: 'border-blue-200 bg-blue-50 text-blue-800',
}

const PRIORITY_STYLES: Record<string, string> = {
  high: 'bg-red-100 text-red-700',
  medium: 'bg-amber-100 text-amber-700',
  low: 'bg-gray-100 text-gray-600',
}

function AlertIcon({ severity }: { severity: string }) {
  if (severity === 'critical') return <AlertTriangle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
  if (severity === 'warning') return <AlertTriangle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
  return <Info className="h-3.5 w-3.5 shrink-0 mt-0.5" />
}

function NationalInsightsPanel() {
  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.nationalInsights(),
    queryFn: getNationalInsights,
    staleTime: 5 * 60 * 1000,
  })

  return (
    <Card className="rounded-2xl border-gray-100 shadow-sm">
      <CardHeader>
        <div className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-indigo-500" />
          <CardTitle className="font-display text-base font-bold">National Curriculum Intelligence</CardTitle>
          {data && !data.ai_configured && (
            <span className="ml-auto text-xs text-gray-400 italic">AI provider not configured — showing summary</span>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading && (
          <div className="flex items-center gap-2 text-xs text-gray-500">
            <Spinner /> Generating national insights...
          </div>
        )}
        {isError && (
          <Banner tone="error">Could not load national insights.</Banner>
        )}
        {data && (
          <>
            {data.headline && (
              <p className="text-sm font-semibold text-gray-900 leading-snug">{data.headline}</p>
            )}
            {data.trend_summary && (
              <p className="text-xs text-gray-600 leading-relaxed">{data.trend_summary}</p>
            )}

            {data.alerts.length > 0 && (
              <div className="space-y-2">
                <p className="text-xs font-semibold uppercase tracking-wide text-gray-400">Alerts</p>
                <div className="space-y-1.5">
                  {data.alerts.map((alert: InsightAlert, i: number) => (
                    <div
                      key={i}
                      className={`flex items-start gap-2 rounded-lg border px-3 py-2 text-xs ${SEVERITY_STYLES[alert.severity] ?? SEVERITY_STYLES.info}`}
                    >
                      <AlertIcon severity={alert.severity} />
                      <span>{alert.message}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {data.recommendations.length > 0 && (
              <div className="space-y-2">
                <p className="text-xs font-semibold uppercase tracking-wide text-gray-400">Recommendations</p>
                <div className="space-y-1.5">
                  {data.recommendations.map((rec: InsightRecommendation, i: number) => (
                    <div key={i} className="flex items-start gap-2 rounded-lg bg-gray-50 px-3 py-2 text-xs text-gray-700">
                      <Lightbulb className="h-3.5 w-3.5 shrink-0 mt-0.5 text-indigo-400" />
                      <div className="flex-1">
                        <span>{rec.text}</span>
                        <span className={`ml-2 inline-flex rounded-full px-1.5 py-0.5 text-[10px] font-bold ${PRIORITY_STYLES[rec.priority] ?? PRIORITY_STYLES.medium}`}>
                          {rec.priority}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {data.alerts.length === 0 && data.recommendations.length === 0 && !isLoading && (
              <p className="text-xs text-gray-400 italic">
                No alerts or recommendations at this time. Configure an LLM provider in the ai-service for AI-generated analysis.
              </p>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
