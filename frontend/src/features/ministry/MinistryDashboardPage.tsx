import { useQueries, useQuery } from '@tanstack/react-query'
import { GraduationCap, Landmark, School as SchoolIcon, Users } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Card, CardContent, CardHeader, CardTitle, Spinner } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { getMinistryOverview, getRegionStats, listRegions } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { formatNumber, formatPercent } from '@lib/utils/format'

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

  const tiles = overview.data
    ? [
        { label: 'Regions', value: overview.data.total_regions, icon: Landmark },
        { label: 'Schools', value: overview.data.total_schools, icon: SchoolIcon },
        { label: 'Students', value: overview.data.total_students, icon: GraduationCap },
        { label: 'Teachers', value: overview.data.total_teachers, icon: Users },
      ]
    : []

  return (
    <AppShell title="Ministry overview" description="National curriculum-intelligence coverage, region by region.">
      <div className="space-y-8">
        {overview.isLoading && (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <Spinner /> Loading national overview…
          </div>
        )}
        {overview.isError && (
          <Banner tone="error">{apiErrorMessage(overview.error, 'Could not load the national overview.')}</Banner>
        )}
        {tiles.length > 0 && (
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
          </div>
        )}

        <Card>
          <CardHeader>
            <CardTitle>Regions</CardTitle>
          </CardHeader>
          <CardContent className="overflow-x-auto p-0">
            {regions.isLoading && (
              <div className="flex items-center gap-2 p-6 text-sm text-gray-500">
                <Spinner /> Loading regions…
              </div>
            )}
            {regions.data && regions.data.length > 0 && (
              <table className="w-full min-w-[640px] text-sm">
                <thead>
                  <tr className="border-b border-gray-200 text-left text-xs uppercase tracking-wide text-gray-400">
                    <th className="px-6 py-2 font-medium">Region</th>
                    <th className="px-3 py-2 font-medium">Schools</th>
                    <th className="px-3 py-2 font-medium">Students</th>
                    <th className="px-3 py-2 font-medium">Teachers</th>
                    <th className="px-6 py-2 font-medium">Avg. score</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {regions.data.map((region, i) => {
                    const stats = regionStats[i]?.data
                    return (
                      <tr key={region.id}>
                        <td className="px-6 py-2 font-medium text-gray-900">
                          {region.name} <span className="text-gray-400">({region.code})</span>
                        </td>
                        <td className="px-3 py-2 font-mono text-gray-600">
                          {stats ? formatNumber(stats.school_count) : '…'}
                        </td>
                        <td className="px-3 py-2 font-mono text-gray-600">
                          {stats ? formatNumber(stats.student_count) : '…'}
                        </td>
                        <td className="px-3 py-2 font-mono text-gray-600">
                          {stats ? formatNumber(stats.teacher_count) : '…'}
                        </td>
                        <td className="px-6 py-2 font-mono text-gray-600">
                          {stats ? formatPercent(stats.avg_assessment_score) : '…'}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            )}
          </CardContent>
        </Card>
      </div>
    </AppShell>
  )
}
