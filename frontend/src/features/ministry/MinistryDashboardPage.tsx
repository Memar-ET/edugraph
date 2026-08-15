import { useQueries, useQuery } from '@tanstack/react-query'
import { AlertTriangle, GraduationCap, Info, Landmark, Lightbulb, School as SchoolIcon, Sparkles, Users } from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  DistributionDonutChart,
  ManagementTableCard,
  PerformanceAreaChart,
  ScheduleCalendarWidget,
  StatMetricCard,
} from '@components/dashboard'
import { Banner, Card, CardContent, CardHeader, CardTitle, Spinner } from '@components/ui'
import { getNationalInsights, getMinistryOverview, getRegionStats, listRegions } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { formatNumber, formatPercent } from '@lib/utils/format'
import type { InsightAlert, InsightRecommendation } from '@/types/api'

const MOCK_MINISTRY_PERFORMANCE = [
  { label: 'Jan', value: 72 },
  { label: 'Feb', value: 76 },
  { label: 'Mar', value: 80 },
  { label: 'Apr', value: 84 },
  { label: 'May', value: 88 },
  { label: 'June', value: 93 },
]

const MOCK_MINISTRY_DONUT = [
  { name: 'National Mastery', value: 78, color: '#2d2d2e' },
  { name: 'Regional Gaps', value: 16, color: '#6b7280' },
  { name: 'Pending Resync', value: 6, color: '#e5e7eb' },
]

const MOCK_MINISTRY_SCHEDULE = [
  {
    id: 'mn1',
    title: 'National Curriculum Oversight',
    time: '10:00 AM - 01:00 PM',
    subtitle: 'Ministry of Education Admin',
    category: 'schedule' as const,
  },
  {
    id: 'mn2',
    title: 'Neo4j Prerequisite Resync',
    time: '04:00 PM (Today)',
    subtitle: '11 Ethiopian Regions',
    category: 'upcoming' as const,
  },
]

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
      schoolCount: stats ? formatNumber(stats.school_count) : '18',
      studentCount: stats ? formatNumber(stats.student_count) : '4,250',
      teacherCount: stats ? formatNumber(stats.teacher_count) : '210',
      avgScore: stats ? formatPercent(stats.avg_assessment_score) : '84.2%',
    }
  }) ?? [
    { id: 'r1', name: 'Addis Ababa', code: 'AA', schoolCount: '42', studentCount: '12,400', teacherCount: '620', avgScore: '86.4%' },
    { id: 'r2', name: 'Oromia', code: 'OR', schoolCount: '128', studentCount: '34,200', teacherCount: '1,450', avgScore: '81.2%' },
    { id: 'r3', name: 'Amhara', code: 'AM', schoolCount: '96', studentCount: '28,100', teacherCount: '1,120', avgScore: '83.0%' },
    { id: 'r4', name: 'Sidama', code: 'SD', schoolCount: '34', studentCount: '9,800', teacherCount: '410', avgScore: '82.5%' },
  ]

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
            <PerformanceAreaChart
              title="National Student Performance"
              subtitle="Monthly Average Mastery Score Across All Regions"
              data={MOCK_MINISTRY_PERFORMANCE}
            />
          </div>

          <div className="lg:col-span-4">
            <DistributionDonutChart
              title="National Quality Distribution"
              centerPercentage="78%"
              centerLabel="National Avg"
              totalValue="83.8% Overall"
              segments={MOCK_MINISTRY_DONUT}
              dateLabel="National Scale"
            />
          </div>

          <div className="lg:col-span-3">
            <ScheduleCalendarWidget
              monthLabel="April 2026"
              scheduleItems={MOCK_MINISTRY_SCHEDULE}
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
