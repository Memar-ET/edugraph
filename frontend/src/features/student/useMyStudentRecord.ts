import { useQuery } from '@tanstack/react-query'

import { listStudents } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'

/** There's no GET /students/me -- career match endpoints take the
 * students.id row id, not the user id. Resolve it once from the school
 * roster (the student role can read /students without a role gate) and
 * cache it alongside the roster query. */
export function useMyStudentRecord() {
  const user = useAuthStore((s) => s.user)
  const schoolId = user?.school_id

  const query = useQuery({
    queryKey: queryKeys.students(schoolId ?? 'unknown'),
    queryFn: () => listStudents(schoolId as string),
    enabled: Boolean(schoolId && user),
  })

  const record = query.data?.find((s) => s.user_id === user?.id)

  return { ...query, record }
}
