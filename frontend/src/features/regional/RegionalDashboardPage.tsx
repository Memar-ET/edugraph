import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, ChevronDown, ChevronUp, GraduationCap, School as SchoolIcon, ShieldCheck, Users } from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  DistributionDonutChart,
  PerformanceAreaChart,
  ScheduleCalendarWidget,
  StatMetricCard,
} from '@components/dashboard'
import { Banner, Card, CardContent, CardHeader, CardTitle, EmptyState, Spinner } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { getRegionStats, getUnderperformingSchools } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { formatNumber, formatPercent } from '@lib/utils/format'
import { useAuthStore } from '@stores/auth.store'
import type { UnderperformingSchool } from '@/types/api'

const MOCK_REGIONAL_PERFORMANCE = [
  { label: 'Jan', value: 68 },
  { label: 'Feb', value: 74 },
  { label: 'Mar', value: 79 },
  { label: 'Apr', value: 83 },
  { label: 'May', value: 86 },
  { label: 'June', value: 91 },
]

const MOCK_REGIONAL_DONUT = [
  { name: 'High Performing', value: 64, color: '#2d2d2e' },
  { name: 'On Track', value: 24, color: '#6b7280' },
  { name: 'Under Audit', value: 12, color: '#e5e7eb' },
]

const MOCK_REGIONAL_SCHEDULE = [
  {
    id: 'rg1',
    title: 'Regional School Audit Meeting',
    time: '09:30 AM - 11:30 AM',
    subtitle: 'Addis Ababa Zonal Board',
    category: 'schedule' as const,
  },
  {
    id: 'rg2',
    title: 'Curriculum Alignment Inspection',
    time: '02:00 PM (Tomorrow)',
    subtitle: '12 Target Schools',
    category: 'upcoming' as const,
  },
]

export function RegionalDashboardPage() {
  const user = useAuthStore((s) => s.user)
  const regionId = user?.region_id

  if (!regionId) {
    return (
      <AppShell title="Regional Dashboard 👋">
        <Banner tone="error">Your account isn&apos;t linked to a region yet. Contact the ministry admin.</Banner>
      </AppShell>
    )
  }

  return (
    <AppShell title="Regional Education Dashboard 👋" description="Schools, enrollment, and composite quality indices across your region.">
      <div className="space-y-6">
        <StatsSection regionId={regionId} />

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
          <div className="lg:col-span-5">
            <PerformanceAreaChart
              title="Regional Assessment Performance"
              subtitle="Monthly Average Mastery Score Across Schools"
              data={MOCK_REGIONAL_PERFORMANCE}
            />
          </div>

          <div className="lg:col-span-4">
            <DistributionDonutChart
              title="School Quality Distribution"
              centerPercentage="83%"
              centerLabel="Avg Score"
              totalValue="83.5% Mastery"
              segments={MOCK_REGIONAL_DONUT}
              dateLabel="Active Region"
            />
          </div>

          <div className="lg:col-span-3">
            <ScheduleCalendarWidget
              monthLabel="April 2026"
              scheduleItems={MOCK_REGIONAL_SCHEDULE}
            />
          </div>
        </div>

        <UnderperformingSection regionId={regionId} />
      </div>
    </AppShell>
  )
}

function StatsSection({ regionId }: { regionId: string }) {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.regionStats(regionId),
    queryFn: () => getRegionStats(regionId),
  })

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 py-4 text-xs text-gray-500">
        <Spinner /> Loading region statistics...
      </div>
    )
  }
  if (isError) return <Banner tone="error">{apiErrorMessage(error, 'Could not load region stats.')}</Banner>

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <StatMetricCard
        title="Total Schools"
        value={formatNumber(data?.school_count ?? 18)}
        change="6.2%"
        trend="up"
        periodText="Registered"
        icon={SchoolIcon}
      />
      <StatMetricCard
        title="Total Students"
        value={formatNumber(data?.student_count ?? 4250)}
        change="11.4%"
        trend="up"
        periodText="Enrolled"
        icon={GraduationCap}
      />
      <StatMetricCard
        title="Total Teachers"
        value={formatNumber(data?.teacher_count ?? 210)}
        change="4.8%"
        trend="up"
        periodText="Staff Roster"
        icon={Users}
      />
      <StatMetricCard
        title="Avg Assessment Score"
        value={formatPercent(data?.avg_assessment_score ?? 83.5)}
        change="8.6%"
        trend="up"
        periodText="Region Index"
        icon={ShieldCheck}
      />
    </div>
  )
}

function UnderperformingSection({ regionId }: { regionId: string }) {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.regionUnderperforming(regionId),
    queryFn: () => getUnderperformingSchools(regionId, 10),
  })

  return (
    <Card className="rounded-2xl border-gray-100 shadow-sm">
      <CardHeader>
        <div className="flex items-center gap-2">
          <AlertTriangle className="h-4 w-4 text-amber-500" />
          <CardTitle className="font-display text-base font-bold">Underperforming Schools</CardTitle>
        </div>
        <p className="text-xs text-gray-500">
          Ranked by average mastery rate — lowest first. Expand a school to see its weakest topics.
        </p>
      </CardHeader>
      <CardContent>
        {isLoading && (
          <div className="flex items-center gap-2 py-4 text-xs text-gray-500">
            <Spinner /> Loading underperforming schools...
          </div>
        )}
        {isError && <Banner tone="error">{apiErrorMessage(error, 'Could not load underperforming schools.')}</Banner>}
        {data && data.schools.length === 0 && (
          <EmptyState title="No school data available yet for this region." />
        )}
        {data && data.schools.length > 0 && (
          <div className="divide-y divide-gray-100">
            {data.schools.map((school, idx) => (
              <SchoolRow key={school.school_id} school={school} rank={idx + 1} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function SchoolRow({ school, rank }: { school: UnderperformingSchool; rank: number }) {
  const [expanded, setExpanded] = useState(false)

  const masteryColor =
    school.mastery_rate < 0.5
      ? 'bg-red-50 text-red-700'
      : school.mastery_rate < 0.7
        ? 'bg-amber-50 text-amber-700'
        : 'bg-emerald-50 text-emerald-700'

  return (
    <div className="py-3">
      <button
        className="flex w-full items-center gap-3 text-left"
        onClick={() => setExpanded((v) => !v)}
        aria-expanded={expanded}
      >
        <span className="w-6 text-center text-xs font-bold text-gray-400">#{rank}</span>
        <div className="flex-1 min-w-0">
          <p className="truncate text-sm font-semibold text-gray-900">{school.school_name}</p>
          <p className="text-xs text-gray-500">{school.school_code}</p>
        </div>
        <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-bold ${masteryColor}`}>
          {formatPercent(school.mastery_rate * 100)}
        </span>
        <span className="text-xs text-gray-400">
          {school.flagged_topics_count} flagged topic{school.flagged_topics_count !== 1 ? 's' : ''}
        </span>
        {expanded ? (
          <ChevronUp className="h-4 w-4 text-gray-400 shrink-0" />
        ) : (
          <ChevronDown className="h-4 w-4 text-gray-400 shrink-0" />
        )}
      </button>

      {expanded && (
        <div className="mt-2 pl-9">
          {school.top_weak_topics.length === 0 ? (
            <p className="text-xs text-gray-400">No topic breakdown available.</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {school.top_weak_topics.map((t) => (
                <span
                  key={t.topic_id}
                  className="inline-flex flex-col rounded-lg border border-red-100 bg-red-50 px-2.5 py-1.5 text-xs"
                  title={`${t.affected_students} students affected`}
                >
                  <span className="font-medium text-red-800 leading-tight">{t.topic_title}</span>
                  <span className="text-red-500 leading-tight">
                    {formatPercent(t.avg_mastery * 100)} avg · {t.affected_students} students
                  </span>
                </span>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
