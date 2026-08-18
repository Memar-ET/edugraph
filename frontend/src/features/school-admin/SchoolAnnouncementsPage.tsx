import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Bell, CheckCheck } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { listNotifications, markNotificationRead } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function SchoolAnnouncementsPage() {
  const qc = useQueryClient()

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.notifications(),
    queryFn: listNotifications,
    refetchInterval: 30_000,
  })

  const markRead = useMutation({
    mutationFn: markNotificationRead,
    onSuccess: () => void qc.invalidateQueries({ queryKey: queryKeys.notifications() }),
  })

  const notifications = data ?? []
  const unread = notifications.filter((n) => !n.is_read)

  return (
    <AppShell title="Announcements & Notifications" description="System notifications and ministry announcements for your school.">
      <div className="space-y-5">
        {unread.length > 0 && (
          <Banner tone="info">
            You have {unread.length} unread notification{unread.length === 1 ? '' : 's'}.
          </Banner>
        )}

        {isLoading && (
          <div className="flex items-center gap-2 py-10 text-xs text-slate-500">
            <Spinner /> Loading announcements...
          </div>
        )}
        {isError && <Banner tone="error">Could not load announcements. Please try again.</Banner>}
        {!isLoading && notifications.length === 0 && (
          <EmptyState
            icon={Bell}
            title="No announcements"
            description="System notifications and ministry alerts will appear here."
          />
        )}
        <div className="space-y-3">
          {notifications.map((n) => (
            <div
              key={n.id}
              className={`rounded-2xl border p-5 shadow-sm transition-colors ${
                n.is_read ? 'border-slate-100 bg-white' : 'border-teal-200 bg-teal-50/30'
              }`}
            >
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    {!n.is_read && <span className="h-2 w-2 rounded-full bg-teal-500 shrink-0" />}
                    <h3 className="font-bold text-sm text-slate-900 truncate">{n.title}</h3>
                  </div>
                  <p className="mt-1 text-xs text-slate-600 leading-relaxed">{n.body}</p>
                  <p className="mt-2 text-[11px] text-slate-400 font-mono">
                    {new Date(n.created_at).toLocaleString()}
                  </p>
                </div>
                {!n.is_read && (
                  <button
                    type="button"
                    onClick={() => markRead.mutate(n.id)}
                    disabled={markRead.isPending}
                    className="shrink-0 flex items-center gap-1 rounded-xl border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50 disabled:opacity-50"
                  >
                    <CheckCheck className="h-3.5 w-3.5" />
                    Mark read
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </AppShell>
  )
}
