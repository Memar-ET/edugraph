import { Bell, CheckCheck } from 'lucide-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { listNotifications, markNotificationRead } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function TeacherAnnouncementsPage() {
  const qc = useQueryClient()

  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.notifications(),
    queryFn: listNotifications,
    refetchInterval: 60_000,
  })

  const markReadMut = useMutation({
    mutationFn: markNotificationRead,
    onSuccess: () => void qc.invalidateQueries({ queryKey: queryKeys.notifications() }),
  })

  const notifications = data ?? []

  return (
    <AppShell
      title="Announcements & Notifications"
      description="System notifications, exam alerts, and platform updates for your account."
    >
      <div className="space-y-4">
        {isLoading && (
          <div className="flex items-center gap-2 py-10 text-xs text-slate-500">
            <Spinner /> Loading announcements…
          </div>
        )}
        {isError && (
          <Banner tone="error">{(error as Error).message ?? 'Could not load announcements.'}</Banner>
        )}
        {!isLoading && notifications.length === 0 && (
          <EmptyState
            title="No announcements"
            description="You'll see system notifications and platform alerts here."
            icon={Bell}
          />
        )}
        {notifications.map((item) => (
          <div
            key={item.id}
            className={`rounded-xl border bg-white p-5 shadow-sm transition-colors ${
              item.is_read ? 'border-slate-100' : 'border-teal-200 bg-teal-50/30'
            }`}
          >
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-2">
                {!item.is_read && (
                  <span className="h-2 w-2 rounded-full bg-teal-500 flex-shrink-0" />
                )}
                <span className="font-bold text-sm text-slate-900">{item.title}</span>
              </div>
              <span className="text-xs text-slate-400 flex-shrink-0 ml-4">
                {new Date(item.created_at).toLocaleDateString('en-US', {
                  month: 'short',
                  day: 'numeric',
                  year: 'numeric',
                })}
              </span>
            </div>

            <p className="mt-1 text-xs text-slate-600 leading-relaxed">{item.body}</p>

            {!item.is_read && (
              <div className="mt-3 pt-3 border-t border-slate-100 flex justify-end">
                <button
                  type="button"
                  disabled={markReadMut.isPending}
                  onClick={() => markReadMut.mutate(item.id)}
                  className="flex items-center gap-1 text-xs font-semibold text-teal-700 hover:underline disabled:opacity-50"
                >
                  <CheckCheck className="h-3.5 w-3.5" />
                  Mark as read
                </button>
              </div>
            )}
          </div>
        ))}
      </div>
    </AppShell>
  )
}
