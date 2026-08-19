import { useQuery } from '@tanstack/react-query'

import { AppShell } from '@components/layout'
import { Banner, Card, CardContent, EmptyState, Spinner } from '@components/ui'
import { getMinistryOverview } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function RegionalAnalyticsPage() {
  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.ministryOverview(),
    queryFn: getMinistryOverview,
  })

  const kpis = [
    { label: 'Total schools', value: data?.total_schools ?? '—' },
    { label: 'Total students', value: data?.total_students ?? '—' },
    { label: 'Regions', value: data?.total_regions ?? '—' },
    { label: 'Total teachers', value: data?.total_teachers ?? '—' },
  ]

  return (
    <AppShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Regional Analytics</h1>
          <p className="mt-1 text-sm text-gray-500">
            Aggregate performance across all schools in your region.
          </p>
        </div>

        {isError && <Banner variant="error" title="Failed to load analytics" description="Try refreshing." />}

        {isLoading ? (
          <div className="flex justify-center py-16"><Spinner /></div>
        ) : (
          <>
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
              {kpis.map((k) => (
                <Card key={k.label}>
                  <CardContent className="py-5 text-center">
                    <p className="text-3xl font-bold text-primary-700">{k.value}</p>
                    <p className="mt-1 text-xs text-gray-500">{k.label}</p>
                  </CardContent>
                </Card>
              ))}
            </div>

            {(!data || Object.keys(data).length === 0) && (
              <EmptyState
                title="No regional data available"
                description="Ministry overview data will appear here once schools have been assessed."
              />
            )}
          </>
        )}
      </div>
    </AppShell>
  )
}
