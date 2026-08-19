import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Search } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Card, CardContent, CardHeader, CardTitle, EmptyState, Spinner } from '@components/ui'
import { listRegions } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function RegionManagementPage() {
  const [search, setSearch] = useState('')

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.regions(),
    queryFn: () => listRegions(),
  })

  const items = (data ?? []).filter(
    (r) => !search || r.name.toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <AppShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Region Management</h1>
          <p className="mt-1 text-sm text-gray-500">
            National overview of all regions and their school counts.
          </p>
        </div>

        {isError && <Banner variant="error" title="Failed to load regions" description="Try refreshing." />}

        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <CardTitle>All Regions</CardTitle>
              <div className="relative ml-auto">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" aria-hidden />
                <input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search regions…"
                  className="rounded-lg border border-gray-200 py-1.5 pl-9 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>
            </div>
          </CardHeader>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="flex justify-center py-12"><Spinner /></div>
            ) : items.length === 0 ? (
              <EmptyState title="No regions found" description="Try a different search." />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-100 bg-gray-50 text-left text-xs font-medium text-gray-500">
                      <th className="px-4 py-3">Region</th>
                      <th className="px-4 py-3">Code</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-50">
                    {items.map((region) => (
                      <tr key={region.id} className="hover:bg-gray-50">
                        <td className="px-4 py-3 font-medium text-gray-900">{region.name}</td>
                        <td className="px-4 py-3 text-gray-500 font-mono text-xs">{region.code ?? '—'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </AppShell>
  )
}
