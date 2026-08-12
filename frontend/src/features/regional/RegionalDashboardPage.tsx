import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { GraduationCap, School as SchoolIcon, ShieldCheck, Users } from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  DistributionDonutChart,
  PerformanceAreaChart,
  ScheduleCalendarWidget,
  StatMetricCard,
} from '@components/dashboard'
import { Banner, Card, CardContent, CardHeader, CardTitle, EmptyState, Select, Spinner } from '@components/ui'
import { QualityScoreGrid } from '@components/shared'
import { apiErrorMessage } from '@lib/api/client'
import { getRegionStats, getSchoolQualityScores, listSchools } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { formatNumber, formatPercent } from '@lib/utils/format'
import { useAuthStore } from '@stores/auth.store'

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

        <SchoolsSection regionId={regionId} />
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

function SchoolsSection({ regionId }: { regionId: string }) {
  const { data: schools, isLoading } = useQuery({
    queryKey: queryKeys.schools(regionId),
    queryFn: () => listSchools(regionId),
  })
  const [selectedSchoolId, setSelectedSchoolId] = useState('')

  return (
    <Card className="rounded-2xl border-gray-100 shadow-sm">
      <CardHeader>
        <CardTitle className="font-display text-base font-bold">Schools in Region</CardTitle>
        <p className="text-xs text-gray-500">Select a school to inspect its composite quality score breakdown.</p>
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading && (
          <div className="flex items-center gap-2 text-xs text-gray-500">
            <Spinner /> Loading schools list...
          </div>
        )}
        {schools && schools.length === 0 && <EmptyState title="No schools registered in this region yet." />}
        {schools && schools.length > 0 && (
          <Select value={selectedSchoolId} onChange={(e) => setSelectedSchoolId(e.target.value)} className="text-xs max-w-md">
            <option value="">Select a school to review...</option>
            {schools.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name} ({s.code})
              </option>
            ))}
          </Select>
        )}
        {selectedSchoolId && <SchoolQuality schoolId={selectedSchoolId} />}
      </CardContent>
    </Card>
  )
}

function SchoolQuality({ schoolId }: { schoolId: string }) {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.schoolQuality(schoolId),
    queryFn: () => getSchoolQualityScores(schoolId),
  })

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-xs text-gray-500">
        <Spinner /> Loading school quality scores...
      </div>
    )
  }
  if (isError) return <Banner tone="error">{apiErrorMessage(error, 'Could not load quality scores.')}</Banner>
  if (!data || data.scores.length === 0) {
    return <EmptyState title="No quality scores computed for this school yet." />
  }
  return <QualityScoreGrid scores={data.scores} />
}
