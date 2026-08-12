import { useQueries, useQuery } from '@tanstack/react-query'
import { GraduationCap, Landmark, School as SchoolIcon, Users } from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  DistributionDonutChart,
  ManagementTableCard,
  PerformanceAreaChart,
  ScheduleCalendarWidget,
  StatMetricCard,
} from '@components/dashboard'
import { getMinistryOverview, getRegionStats, listRegions } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { formatNumber, formatPercent } from '@lib/utils/format'

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
