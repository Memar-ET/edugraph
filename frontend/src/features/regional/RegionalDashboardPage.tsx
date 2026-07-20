import { useQuery } from '@tanstack/react-query'
import { GraduationCap, School as SchoolIcon, ShieldCheck, Users } from 'lucide-react'
import { useState } from 'react'

import { AppShell } from '@components/layout'
import { Banner, Card, CardContent, CardHeader, CardTitle, EmptyState, Select, Spinner } from '@components/ui'
import { QualityScoreGrid } from '@components/shared'
import { apiErrorMessage } from '@lib/api/client'
import { getRegionStats, getSchoolQualityScores, listSchools } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { formatNumber, formatPercent } from '@lib/utils/format'
import { useAuthStore } from '@stores/auth.store'

export function RegionalDashboardPage() {
  const user = useAuthStore((s) => s.user)
  const regionId = user?.region_id

  if (!regionId) {
    return (
      <AppShell title="Regional dashboard">
        <Banner tone="error">Your account isn&apos;t linked to a region yet. Contact the ministry admin.</Banner>
      </AppShell>
    )
  }

  return (
    <AppShell title="Regional dashboard" description="Schools, enrollment, and quality across your region.">
      <div className="space-y-8">
        <StatsSection regionId={regionId} />
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
      <div className="flex items-center gap-2 text-sm text-gray-500">
        <Spinner /> Loading region stats…
      </div>
    )
  }
  if (isError) return <Banner tone="error">{apiErrorMessage(error, 'Could not load region stats.')}</Banner>
  if (!data) return null

  const tiles = [
    { label: 'Schools', value: data.school_count, icon: SchoolIcon },
    { label: 'Students', value: data.student_count, icon: GraduationCap },
    { label: 'Teachers', value: data.teacher_count, icon: Users },
  ]

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {tiles.map((tile) => (
        <Card key={tile.label}>
          <CardContent className="flex items-center gap-3 pt-6">
            <tile.icon className="h-5 w-5 text-primary-700" aria-hidden />
            <div>
              <p className="font-mono text-2xl font-semibold text-gray-900">{formatNumber(tile.value)}</p>
              <p className="text-xs text-gray-500">{tile.label}</p>
            </div>
          </CardContent>
        </Card>
      ))}
      <Card>
        <CardContent className="flex items-center gap-3 pt-6">
          <ShieldCheck className="h-5 w-5 text-primary-700" aria-hidden />
          <div>
            <p className="font-mono text-2xl font-semibold text-gray-900">{formatPercent(data.avg_assessment_score)}</p>
            <p className="text-xs text-gray-500">Avg. assessment score</p>
          </div>
        </CardContent>
      </Card>
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
    <Card>
      <CardHeader>
        <CardTitle>Schools in this region</CardTitle>
        <p className="mt-1 text-sm text-gray-500">Pick a school to see its quality scores.</p>
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <Spinner /> Loading schools…
          </div>
        )}
        {schools && schools.length === 0 && <EmptyState title="No schools registered in this region yet." />}
        {schools && schools.length > 0 && (
          <Select value={selectedSchoolId} onChange={(e) => setSelectedSchoolId(e.target.value)}>
            <option value="">Select a school…</option>
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
      <div className="flex items-center gap-2 text-sm text-gray-500">
        <Spinner /> Loading quality scores…
      </div>
    )
  }
  if (isError) return <Banner tone="error">{apiErrorMessage(error, 'Could not load quality scores.')}</Banner>
  if (!data || data.scores.length === 0) {
    return <EmptyState title="No quality scores computed for this school yet." />
  }
  return <QualityScoreGrid scores={data.scores} />
}
