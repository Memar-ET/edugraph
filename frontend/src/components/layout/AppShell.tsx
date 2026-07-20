import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import { LogOut, Menu, X } from 'lucide-react'
import { useState } from 'react'
import type { ReactNode } from 'react'

import { Button } from '@components/ui'
import { cn } from '@lib/utils/cn'
import { useAuthStore } from '@stores/auth.store'

import { NotificationBell } from './NotificationBell'
import { getNavItems, ROLE_LABELS } from './nav-config'

export interface AppShellProps {
  title: string
  description?: string
  actions?: ReactNode
  children: ReactNode
}

/** Persistent role-scoped shell -- sidebar "ledger index" nav + a document
 * header band -- wrapping every authenticated page in the app. */
export function AppShell({ title, description, actions, children }: AppShellProps) {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const clearAuth = useAuthStore((s) => s.clearAuth)
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const navItems = getNavItems(user?.role)

  const handleLogout = () => {
    clearAuth()
    void navigate({ to: '/login' })
  }

  return (
    <div className="flex min-h-screen bg-gray-50">
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-40 flex w-64 flex-col border-r border-gray-200 bg-white transition-transform duration-200 lg:static lg:translate-x-0',
          mobileNavOpen ? 'translate-x-0' : '-translate-x-full',
        )}
      >
        <div className="flex h-16 items-center gap-2 border-b border-gray-200 px-6">
          <span className="font-display text-lg font-semibold text-primary-800">EduGraph</span>
        </div>
        <nav className="flex-1 space-y-1 px-3 py-4">
          {navItems.map((item) => {
            const active = pathname === item.to
            const Icon = item.icon
            return (
              <Link
                key={item.to}
                // eslint-disable-next-line @typescript-eslint/no-explicit-any -- nav items are role-derived, not statically known route literals
                to={item.to as any}
                onClick={() => setMobileNavOpen(false)}
                className={cn(
                  'flex items-center gap-3 rounded-md border-l-[3px] px-3 py-2 text-sm font-medium transition-colors',
                  active
                    ? 'border-primary-700 bg-primary-50 text-primary-800'
                    : 'border-transparent text-gray-600 hover:bg-gray-100 hover:text-gray-900',
                )}
              >
                <Icon className="h-4 w-4" aria-hidden />
                {item.label}
              </Link>
            )
          })}
        </nav>
        <div className="border-t border-gray-200 p-4">
          <p className="truncate text-sm font-medium text-gray-900">{user?.full_name}</p>
          <p className="text-xs text-gray-500">{user ? ROLE_LABELS[user.role] : ''}</p>
          <Button variant="ghost" size="sm" className="mt-2 w-full justify-start" onClick={handleLogout}>
            <LogOut className="h-4 w-4" aria-hidden />
            Sign out
          </Button>
        </div>
      </aside>

      {mobileNavOpen && (
        <button
          type="button"
          aria-label="Close navigation"
          className="fixed inset-0 z-30 bg-primary-900/30 lg:hidden"
          onClick={() => setMobileNavOpen(false)}
        />
      )}

      <div className="flex min-h-screen flex-1 flex-col">
        <header className="sticky top-0 z-20 flex h-16 items-center justify-between border-b border-gray-200 bg-white/95 px-4 backdrop-blur sm:px-6">
          <div className="flex items-center gap-3">
            <button
              type="button"
              className="rounded-md p-2 text-gray-500 hover:bg-gray-100 lg:hidden"
              onClick={() => setMobileNavOpen((v) => !v)}
              aria-label="Toggle navigation"
            >
              {mobileNavOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
            <div>
              <h1 className="font-display text-xl font-semibold text-gray-900">{title}</h1>
              {description && <p className="text-sm text-gray-500">{description}</p>}
            </div>
          </div>
          <div className="flex items-center gap-2">
            {actions}
            <NotificationBell />
          </div>
        </header>
        <main className="flex-1 px-4 py-6 sm:px-6 lg:px-8">{children}</main>
      </div>
    </div>
  )
}
