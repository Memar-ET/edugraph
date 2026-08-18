import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Search } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Card, CardContent, CardHeader, CardTitle, EmptyState, Pagination, Spinner } from '@components/ui'
import { listAllTeachers } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'

export function TeacherRosterPage() {
  const user = useAuthStore((s) => s.user)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.allTeachers({ school_id: user?.school_id, page }),
    queryFn: () => listAllTeachers({ school_id: user?.school_id, page, limit: 30 }),
    enabled: !!user?.school_id,
  })

  const items = (data?.items ?? []).filter(
    (t) =>
      !search ||
      t.full_name?.toLowerCase().includes(search.toLowerCase()) ||
      t.email?.toLowerCase().includes(search.toLowerCase()),
  )
  const totalPages = Math.ceil((data?.meta?.total ?? 0) / 30)

  return (
    <AppShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Teachers</h1>
          <p className="mt-1 text-sm text-gray-500">All teachers assigned to this school.</p>
        </div>

        {isError && <Banner variant="error" title="Failed to load teachers" description="Try refreshing." />}

        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <CardTitle>Teacher Roster</CardTitle>
              <div className="relative ml-auto">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" aria-hidden />
                <input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search name or email…"
                  className="rounded-lg border border-gray-200 py-1.5 pl-9 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>
            </div>
          </CardHeader>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="flex justify-center py-12"><Spinner /></div>
            ) : items.length === 0 ? (
              <EmptyState title="No teachers found" description="Try a different search term." />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-100 bg-gray-50 text-left text-xs font-medium text-gray-500">
                      <th className="px-4 py-3">Name</th>
                      <th className="px-4 py-3">Email</th>
                      <th className="px-4 py-3">Joined</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-50">
                    {items.map((teacher) => (
                      <tr key={teacher.id} className="hover:bg-gray-50">
                        <td className="px-4 py-3 font-medium text-gray-900">{teacher.full_name ?? '—'}</td>
                        <td className="px-4 py-3 text-gray-600">{teacher.email ?? '—'}</td>
                        <td className="px-4 py-3 text-gray-500">
                          {new Date(teacher.created_at).toLocaleDateString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        {totalPages > 1 && (
          <div className="flex justify-center">
            <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
          </div>
        )}
      </div>
    </AppShell>
  )
}
