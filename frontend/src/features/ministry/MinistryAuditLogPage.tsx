import { useQuery } from '@tanstack/react-query'
import { Bell } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { listNotifications } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function MinistryAuditLogPage() {
  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.notifications(),
    queryFn: listNotifications,
  })

  const entries = data ?? []

  return (
    <AppShell
      title="System Audit & Notifications Log"
      description="System-level notifications, administrative events, and security alerts for ministry administrators."
    >
      <div className="space-y-4">
        {isError && (
          <Banner
            variant="error"
            title="Failed to load audit log"
            description="Try refreshing the page."
          />
        )}

        <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
          {isLoading ? (
            <div className="flex justify-center py-12">
              <Spinner />
            </div>
          ) : entries.length === 0 ? (
            <EmptyState
              icon={Bell}
              title="No audit events"
              description="System notifications and audit events will appear here."
            />
          ) : (
            <table className="w-full text-left text-xs text-slate-600">
              <thead className="border-b border-slate-100 bg-slate-50 text-[11px] font-bold uppercase tracking-wider text-slate-500">
                <tr>
                  <th className="px-4 py-3">Event</th>
                  <th className="px-4 py-3">Detail</th>
                  <th className="px-4 py-3 text-center">Status</th>
                  <th className="px-4 py-3 text-right">Timestamp</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 font-normal">
                {entries.map((entry) => (
                  <tr key={entry.id} className="hover:bg-slate-50/80">
                    <td className="px-4 py-3 font-semibold text-slate-900">{entry.title}</td>
                    <td className="px-4 py-3 max-w-xs truncate text-slate-600">{entry.body}</td>
                    <td className="px-4 py-3 text-center">
                      <span
                        className={`rounded-full px-2.5 py-0.5 text-[10px] font-bold ${
                          entry.is_read
                            ? 'bg-slate-100 text-slate-500'
                            : 'bg-teal-100 text-teal-800'
                        }`}
                      >
                        {entry.is_read ? 'Read' : 'New'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right font-mono text-slate-400">
                      {new Date(entry.created_at).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </AppShell>
  )
}
