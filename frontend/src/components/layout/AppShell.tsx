import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import { Command, Globe, GraduationCap, LogOut, Menu, Search, X } from 'lucide-react'
import { useState } from 'react'
import type { ReactNode } from 'react'

import { logout } from '@lib/api/endpoints'
import { cn } from '@lib/utils/cn'
import { useAuthStore } from '@stores/auth.store'

import { CommandPalette } from '@components/shared/CommandPalette'
import { NotificationBell } from './NotificationBell'
import { getNavItems, NAV_SECTIONS, ROLE_LABELS } from './nav-config'

export interface AppShellProps {
  /** Optional slot for primary page-level action buttons shown in the top bar. */
  actions?: ReactNode
  /** Unused in this layout — kept so existing call sites don't need updating. */
  title?: string
  description?: string
  children: ReactNode
}

export function AppShell({ actions, children }: AppShellProps) {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const clearAuth = useAuthStore((s) => s.clearAuth)
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [lang, setLang] = useState<'EN' | 'AM'>('EN')
  const [commandOpen, setCommandOpen] = useState(false)

  const navItems = getNavItems(user?.role)

  const handleLogout = () => {
    void logout()
    clearAuth()
    void navigate({ to: '/login' })
  }

  const initials =
    user?.full_name
      ?.split(' ')
      .slice(0, 2)
      .map((n) => n[0])
      .join('') ?? 'U'

  return (
    <div className="flex h-screen overflow-hidden bg-gray-50">
      {/* Global Command Palette */}
      <CommandPalette isOpen={commandOpen} onClose={() => setCommandOpen(false)} />

      {/* ── Mobile overlay ──────────────────────────────────────────── */}
      {sidebarOpen && (
        <button
          type="button"
          aria-label="Close navigation"
          className="fixed inset-0 z-30 bg-gray-900/40 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* ── Sidebar ─────────────────────────────────────────────────── */}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-40 flex w-64 flex-col bg-white border-r border-gray-200 transition-transform duration-200 lg:static lg:translate-x-0',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full',
        )}
      >
        {/* Logo */}
        <div className="flex h-14 shrink-0 items-center gap-2.5 border-b border-gray-100 px-5">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-teal-700">
            <GraduationCap className="h-4 w-4 text-white" aria-hidden />
          </div>
          <span className="text-sm font-bold tracking-tight text-gray-900">EduGraph AI</span>
        </div>

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto px-3 py-4" aria-label="Main navigation">
          {NAV_SECTIONS.map((section) => {
            const items = navItems.filter((i) => i.section === section)
            if (!items.length) return null
            return (
              <div key={section} className="mb-5">
                <p className="mb-1.5 px-2.5 text-[10px] font-semibold uppercase tracking-widest text-gray-400">
                  {section}
                </p>
                <div className="space-y-0.5">
                  {items.map((item) => {
                    const active = pathname === item.to
                    const Icon = item.icon
                    return (
                      <Link
                        key={item.to}
                        // eslint-disable-next-line @typescript-eslint/no-explicit-any
                        to={item.to as any}
                        onClick={() => setSidebarOpen(false)}
                        className={cn(
                          'flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm font-medium transition-colors',
                          active
                            ? 'bg-teal-50 text-teal-800 font-bold'
                            : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900',
                        )}
                        aria-current={active ? 'page' : undefined}
                      >
                        <Icon
                          className={cn(
                            'h-4 w-4 shrink-0',
                            active ? 'text-teal-700' : 'text-gray-400',
                          )}
                          aria-hidden
                        />
                        <span className="truncate">{item.label}</span>
                        {item.badge != null && (
                          <span
                            className={cn(
                              'ml-auto flex h-5 min-w-[20px] items-center justify-center rounded-full px-1.5 text-[10px] font-bold',
                              active
                                ? 'bg-teal-100 text-teal-800'
                                : 'bg-gray-100 text-gray-600',
                            )}
                          >
                            {item.badge}
                          </span>
                        )}
                      </Link>
                    )
                  })}
                </div>
              </div>
            )
          })}
        </nav>

        {/* User footer */}
        <div className="shrink-0 border-t border-gray-100 p-3">
          <div className="flex items-center gap-2.5 rounded-lg px-2 py-2">
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-teal-800 text-xs font-bold text-white">
              {initials}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-xs font-semibold text-gray-900">{user?.full_name}</p>
              <p className="truncate text-[11px] text-gray-500">
                {user ? ROLE_LABELS[user.role] : ''}
              </p>
            </div>
            <button
              type="button"
              onClick={handleLogout}
              aria-label="Sign out"
              title="Sign out"
              className="rounded-md p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 focus:outline-none focus:ring-2 focus:ring-teal-500"
            >
              <LogOut className="h-4 w-4" aria-hidden />
            </button>
          </div>
        </div>
      </aside>

      {/* ── Main area ───────────────────────────────────────────────── */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Top bar */}
        <header className="relative flex h-14 shrink-0 items-center gap-3 border-b border-gray-200 bg-white px-4 sm:px-6">
          {/* Mobile sidebar toggle */}
          <button
            type="button"
            aria-label={sidebarOpen ? 'Close menu' : 'Open menu'}
            className="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-teal-500 lg:hidden"
            onClick={() => setSidebarOpen((v) => !v)}
          >
            {sidebarOpen ? (
              <X className="h-5 w-5" aria-hidden />
            ) : (
              <Menu className="h-5 w-5" aria-hidden />
            )}
          </button>

          {/* Command Palette Launcher */}
          <button
            type="button"
            onClick={() => setCommandOpen(true)}
            className="flex items-center gap-2 rounded-xl border border-slate-200 bg-slate-50 py-1.5 px-3 text-xs text-slate-500 hover:bg-white hover:border-slate-300 transition-all w-64"
          >
            <Search className="h-3.5 w-3.5 text-slate-400" />
            <span className="truncate">Type a command or search...</span>
            <span className="ml-auto hidden sm:flex items-center gap-0.5 rounded bg-white px-1.5 py-0.5 text-[10px] font-mono border border-slate-200 text-slate-400">
              <Command className="h-2.5 w-2.5" />K
            </span>
          </button>

          {/* Right-side controls */}
          <div className="ml-auto flex items-center gap-2">
            {/* Page-level actions slot */}
            {actions}

            {/* Language toggle */}
            <button
              type="button"
              onClick={() => setLang((l) => (l === 'EN' ? 'AM' : 'EN'))}
              aria-label={`Switch to ${lang === 'EN' ? 'Amharic' : 'English'}`}
              className="flex items-center gap-1.5 rounded-lg border border-gray-200 px-2.5 py-1.5 text-xs font-semibold text-gray-600 transition-colors hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-teal-500"
            >
              <Globe className="h-3.5 w-3.5" aria-hidden />
              {lang}
            </button>

            {/* Notification bell -- self-contained (own Radix dropdown
                trigger + panel, real API data via listNotifications).
                Previously wrapped in a second <button>, which put a
                <button> inside a <button> (invalid DOM nesting) and paired
                it with NotificationCenter, a fully mock-data component
                (MOCK_NOTIFICATIONS, no API call) -- removed. */}
            <NotificationBell />
          </div>
        </header>

        {/* Scrollable page content */}
        <main className="flex-1 overflow-y-auto p-6">
          {children}
        </main>
      </div>
    </div>
  )
}

