import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Search } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Card, CardContent, CardHeader, CardTitle, EmptyState, Pagination, Spinner } from '@components/ui'
import { listAllStudents } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'

export function StudentRosterPage() {
  const user = useAuthStore((s) => s.user)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.allStudents({ school_id: user?.school_id, page }),
    queryFn: () => listAllStudents({ school_id: user?.school_id, page, limit: 30 }),
    enabled: !!user?.school_id,
  })

  const items = (data?.items ?? []).filter(
    (s) =>
      !search ||
      s.full_name?.toLowerCase().includes(search.toLowerCase()) ||
      s.admission_no.toLowerCase().includes(search.toLowerCase()),
  )
  const totalPages = Math.ceil((data?.meta?.total ?? 0) / 30)

  return (
    <AppShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Students</h1>
          <p className="mt-1 text-sm text-gray-500">All students enrolled at this school.</p>
        </div>

        {isError && <Banner variant="error" title="Failed to load students" description="Try refreshing." />}

        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <CardTitle>Student Roster</CardTitle>
              <div className="relative ml-auto">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" aria-hidden />
                <input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search…"
                  className="rounded-lg border border-gray-200 py-1.5 pl-9 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>
            </div>
          </CardHeader>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="flex justify-center py-12"><Spinner /></div>
            ) : items.length === 0 ? (
              <EmptyState title="No students found" description="Try a different search term." />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-100 bg-gray-50 text-left text-xs font-medium text-gray-500">
                      <th className="px-4 py-3">Name</th>
                      <th className="px-4 py-3">Admission No.</th>
                      <th className="px-4 py-3 text-center">Grade</th>
                      <th className="px-4 py-3">Joined</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-50">
                    {items.map((student) => (
                      <tr key={student.id} className="hover:bg-gray-50">
                        <td className="px-4 py-3 font-medium text-gray-900">{student.full_name ?? '—'}</td>
                        <td className="px-4 py-3 text-gray-600">{student.admission_no}</td>
                        <td className="px-4 py-3 text-center text-gray-600">G{student.grade_level}</td>
                        <td className="px-4 py-3 text-gray-500">
                          {new Date(student.created_at).toLocaleDateString()}
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
