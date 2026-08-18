import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, ChevronDown, ChevronUp, GraduationCap, School as SchoolIcon, ShieldCheck, Users } from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  DistributionDonutChart,
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

export function RegionalDashboardPage() {
  const user = useAuthStore((s) => s.user)
  const regionId = user?.region_id

  const { data: stats } = useQuery({
    queryKey: queryKeys.regionStats(regionId ?? ''),
    queryFn: () => getRegionStats(regionId!),
    enabled: !!regionId,
  })

  const { data: underperforming } = useQuery({
    queryKey: queryKeys.regionUnderperforming(regionId ?? ''),
    queryFn: () => getUnderperformingSchools(regionId!, 10),
    enabled: !!regionId,
  })

  const donutSegments = useMemo(() => {
    const schools = underperforming?.schools ?? []
    if (schools.length === 0) return []
    const high = schools.filter((s) => s.mastery_rate >= 0.8).length
    const mid = schools.filter((s) => s.mastery_rate >= 0.6 && s.mastery_rate < 0.8).length
    const low = schools.filter((s) => s.mastery_rate < 0.6).length
    return [
      { name: 'High Performing', value: high, color: '#2d2d2e' },
      { name: 'On Track', value: mid, color: '#6b7280' },
      { name: 'Under Audit', value: low, color: '#e5e7eb' },
    ].filter((s) => s.value > 0)
  }, [underperforming])

  const scheduleItems = useMemo(() => {
    return (underperforming?.schools ?? []).slice(0, 3).map((s) => ({
      id: s.school_id,
      title: `${s.school_name} — Needs Support`,
      time: `${(s.mastery_rate * 100).toFixed(0)}% mastery · ${s.flagged_topics_count} topics`,
      subtitle: s.school_code,
      category: 'upcoming' as const,
    }))
  }, [underperforming])

  const avgMastery = stats?.avg_assessment_score ?? 0

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
            <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm h-full">
              <h3 className="font-bold text-sm text-slate-900">Region Snapshot</h3>
              <p className="text-xs text-slate-500 mt-0.5">Key indicators at a glance</p>
              <div className="mt-4 grid grid-cols-2 gap-3">
                <div className="rounded-xl bg-slate-50 p-3 border border-slate-100">
                  <p className="text-xs text-slate-500">Schools</p>
                  <p className="text-xl font-bold text-slate-900 mt-0.5">{stats ? formatNumber(stats.school_count) : '—'}</p>
                </div>
                <div className="rounded-xl bg-slate-50 p-3 border border-slate-100">
                  <p className="text-xs text-slate-500">Students</p>
                  <p className="text-xl font-bold text-slate-900 mt-0.5">{stats ? formatNumber(stats.student_count) : '—'}</p>
                </div>
                <div className="rounded-xl bg-slate-50 p-3 border border-slate-100">
                  <p className="text-xs text-slate-500">Teachers</p>
                  <p className="text-xl font-bold text-slate-900 mt-0.5">{stats ? formatNumber(stats.teacher_count) : '—'}</p>
                </div>
                <div className="rounded-xl bg-teal-50 p-3 border border-teal-100">
                  <p className="text-xs text-teal-600">Avg Mastery</p>
                  <p className="text-xl font-bold text-teal-800 mt-0.5">{avgMastery > 0 ? `${avgMastery.toFixed(1)}%` : '—'}</p>
                </div>
              </div>
            </div>
          </div>

          <div className="lg:col-span-4">
            <DistributionDonutChart
              title="School Quality Distribution"
              centerPercentage={avgMastery > 0 ? `${avgMastery.toFixed(0)}%` : '—'}
              centerLabel="Avg Mastery"
              totalValue={stats ? `${stats.school_count} Schools` : 'Loading…'}
              segments={donutSegments}
              dateLabel="Active Region"
            />
          </div>

          <div className="lg:col-span-3">
            <ScheduleCalendarWidget
              monthLabel={new Date().toLocaleDateString(undefined, { month: 'long', year: 'numeric' })}
              scheduleItems={scheduleItems}
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
