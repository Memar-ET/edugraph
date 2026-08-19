import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Search } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Card, CardContent, CardHeader, CardTitle, EmptyState, Spinner } from '@components/ui'
import { listSchools } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function RegionalSchoolsPage() {
  const [search, setSearch] = useState('')

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.schools(),
    queryFn: () => listSchools(),
  })

  const items = (data ?? []).filter(
    (s) => !search || s.name.toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <AppShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Schools in Region</h1>
          <p className="mt-1 text-sm text-gray-500">
            Overview of all schools under your regional administration.
          </p>
        </div>

        {isError && <Banner variant="error" title="Failed to load schools" description="Try refreshing." />}

        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <CardTitle>School List</CardTitle>
              <div className="relative ml-auto">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" aria-hidden />
                <input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search schools…"
                  className="rounded-lg border border-gray-200 py-1.5 pl-9 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>
            </div>
          </CardHeader>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="flex justify-center py-12"><Spinner /></div>
            ) : items.length === 0 ? (
              <EmptyState title="No schools found" description="Try a different search." />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-100 bg-gray-50 text-left text-xs font-medium text-gray-500">
                      <th className="px-4 py-3">School</th>
                      <th className="px-4 py-3">Location</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-50">
                    {items.map((school) => (
                      <tr key={school.id} className="hover:bg-gray-50">
                        <td className="px-4 py-3 font-medium text-gray-900">{school.name}</td>
                        <td className="px-4 py-3 text-gray-500">{school.address ?? '—'}</td>
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
