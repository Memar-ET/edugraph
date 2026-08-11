import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Bell } from 'lucide-react'

import { listNotifications, markNotificationRead } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { cn } from '@lib/utils/cn'
import { formatRelative } from '@lib/utils/date'

export function NotificationBell() {
  const queryClient = useQueryClient()
  const { data: notifications = [] } = useQuery({
    queryKey: queryKeys.notifications(),
    queryFn: listNotifications,
    refetchInterval: 60_000,
  })
  const unreadCount = notifications.filter((n) => !n.is_read).length

  const handleRead = (id: string) => {
    void markNotificationRead(id).then(() => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications() })
    })
  }

  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button
          type="button"
          className="relative rounded-md p-2 text-gray-500 hover:bg-gray-100 hover:text-gray-900"
          aria-label={unreadCount > 0 ? `Notifications, ${unreadCount} unread` : 'Notifications'}
        >
          <Bell className="h-5 w-5" aria-hidden />
          {unreadCount > 0 && (
            <span className="absolute right-1 top-1 flex h-4 w-4 items-center justify-center rounded-full bg-alert-500 text-[10px] font-semibold text-white">
              {unreadCount > 9 ? '9+' : unreadCount}
            </span>
          )}
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          align="end"
          sideOffset={8}
          className="z-50 w-80 rounded-lg border border-gray-200 bg-white p-1 shadow-lg"
        >
          <div className="border-b border-gray-100 px-3 py-2 text-sm font-semibold text-gray-900">
            Notifications
          </div>
          {notifications.length === 0 ? (
            <p className="px-3 py-6 text-center text-sm text-gray-500">You&apos;re all caught up.</p>
          ) : (
            <div className="max-h-96 overflow-y-auto">
              {notifications.map((n) => (
                <DropdownMenu.Item
                  key={n.id}
                  onSelect={() => {
                    if (!n.is_read) handleRead(n.id)
                  }}
                  className={cn(
                    'flex cursor-pointer flex-col gap-0.5 rounded-md px-3 py-2 text-sm outline-none focus:bg-gray-50',
                    !n.is_read && 'bg-primary-50/60',
                  )}
                >
                  <span className="flex items-center gap-2 font-medium text-gray-900">
                    {!n.is_read && <span className="h-1.5 w-1.5 rounded-full bg-primary-600" aria-hidden />}
                    {n.title}
                  </span>
                  <span className="text-gray-600">{n.body}</span>
                  <span className="text-xs text-gray-400">{formatRelative(n.created_at)}</span>
                </DropdownMenu.Item>
              ))}
            </div>
          )}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
