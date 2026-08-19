import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Search } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Card, CardContent, CardHeader, CardTitle, EmptyState, Pagination, Spinner } from '@components/ui'
import { listAllStudents } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'

export function TeacherStudentRosterPage() {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.allStudents({ school_id: user?.school_id }),
    queryFn: () => listAllStudents({ school_id: user?.school_id, page, limit: 30 }),
    enabled: !!user?.school_id,
  })

  const items = (data?.items ?? []).filter((s) =>
    !search ||
    s.full_name?.toLowerCase().includes(search.toLowerCase()) ||
    s.email?.toLowerCase().includes(search.toLowerCase()) ||
    s.admission_no.toLowerCase().includes(search.toLowerCase()),
  )

  const totalPages = Math.ceil((data?.meta?.total ?? 0) / 30)

  return (
    <AppShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">My Students</h1>
          <p className="mt-1 text-sm text-gray-500">
            Students enrolled at your school. Click a row to view their skill states and exam history.
          </p>
        </div>

        {isError && (
          <Banner tone="error" title="Failed to load students" description="Try refreshing the page." />
        )}

        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <CardTitle>Student Roster</CardTitle>
              <div className="relative ml-auto">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" aria-hidden />
                <input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search name or ID…"
                  className="rounded-lg border border-gray-200 py-1.5 pl-9 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>
            </div>
          </CardHeader>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="flex justify-center py-12">
                <Spinner />
              </div>
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
                      <tr
                        key={student.id}
                        onClick={() =>
                          navigate({
                            to: '/teacher/students/$studentId',
                            params: { studentId: student.id },
                          })
                        }
                        className="cursor-pointer hover:bg-gray-50"
                      >
                        <td className="px-4 py-3 font-medium text-gray-900">
                          {student.full_name ?? '—'}
                        </td>
                        <td className="px-4 py-3 text-gray-600">{student.admission_no}</td>
                        <td className="px-4 py-3 text-center text-gray-600">
                          G{student.grade_level}
                        </td>
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
