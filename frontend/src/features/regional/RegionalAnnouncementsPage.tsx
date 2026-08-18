import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Bell, CheckCheck } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { listNotifications, markNotificationRead } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function RegionalAnnouncementsPage() {
  const qc = useQueryClient()

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.notifications(),
    queryFn: listNotifications,
  })

  const markReadMutation = useMutation({
    mutationFn: (id: string) => markNotificationRead(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: queryKeys.notifications() }),
  })

  const notifications = data ?? []
  const unread = notifications.filter((n) => !n.is_read).length

  return (
    <AppShell
      title="Regional Directives & Bulletins"
      description="System notifications and regional policy bulletins from the Ministry of Education."
    >
      <div className="space-y-4">
        {isError && (
          <Banner variant="error" title="Failed to load notifications" description="Try refreshing." />
        )}

        {unread > 0 && (
          <div className="rounded-xl border border-teal-200 bg-teal-50 px-4 py-3 text-xs text-teal-800 font-medium">
            {unread} unread notification{unread !== 1 ? 's' : ''}
          </div>
        )}

        {isLoading ? (
          <div className="flex justify-center py-16"><Spinner /></div>
        ) : notifications.length === 0 ? (
          <EmptyState
            icon={Bell}
            title="No directives"
            description="Regional bulletins and directives will appear here."
          />
        ) : (
          <div className="space-y-3">
            {notifications.map((n) => (
              <div
                key={n.id}
                className={`rounded-xl border p-5 shadow-sm ${
                  n.is_read ? 'border-slate-200 bg-white' : 'border-teal-200 bg-teal-50/40'
                }`}
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="space-y-1">
                    <h3 className="text-sm font-bold text-slate-900">{n.title}</h3>
                    <p className="text-xs text-slate-600 leading-relaxed">{n.body}</p>
                    <p className="text-[10px] font-mono text-slate-400">
                      {new Date(n.created_at).toLocaleString()}
                    </p>
                  </div>
                  {!n.is_read && (
                    <button
                      type="button"
                      onClick={() => markReadMutation.mutate(n.id)}
                      disabled={markReadMutation.isPending}
                      className="shrink-0 flex items-center gap-1 rounded-lg border border-slate-200 px-2.5 py-1 text-xs font-medium text-slate-600 hover:bg-slate-50"
                    >
                      <CheckCheck className="h-3.5 w-3.5" /> Mark read
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </AppShell>
  )
}
