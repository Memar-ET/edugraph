import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import {
  ChevronDown,
  GraduationCap,
  LogOut,
  Menu,
  MessageSquare,
  Search,
  X,
} from 'lucide-react'
import { useState } from 'react'
import type { ReactNode } from 'react'

import { cn } from '@lib/utils/cn'
import { useAuthStore } from '@stores/auth.store'

import { NotificationBell } from './NotificationBell'
import { getNavItems, ROLE_LABELS } from './nav-config'

export interface AppShellProps {
  title?: string
  description?: string
  actions?: ReactNode
  children: ReactNode
}

export function AppShell({ title, description, actions, children }: AppShellProps) {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const clearAuth = useAuthStore((s) => s.clearAuth)
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')

  const navItems = getNavItems(user?.role)

  const handleLogout = () => {
    // Must revoke server-side too -- clearAuth() alone can't touch the
    // HttpOnly session cookies (checklist 11.1), see endpoints.ts's
    // logout() for why.
    void logout()
    clearAuth()
    void navigate({ to: '/login' })
  }

  const sections = ['General', 'Reports', 'Settings'] as const

  return (
    <div className="flex min-h-screen bg-[#f3f4f6] text-gray-900 antialiased font-sans">
      {/* Sidebar Navigation */}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-40 flex w-64 flex-col border-r border-gray-200/80 bg-white shadow-sm transition-transform duration-200 lg:static lg:translate-x-0',
          mobileNavOpen ? 'translate-x-0' : '-translate-x-full',
        )}
      >
        {/* Brand Header */}
        <div className="flex h-20 items-center gap-3 border-b border-gray-100 px-6">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gray-900 text-white shadow-sm">
            <GraduationCap className="h-5 w-5" />
          </div>
          <div>
            <span className="font-display text-xl font-extrabold tracking-tight text-gray-900">
              EduGraph
            </span>
            <span className="block text-[10px] font-semibold uppercase tracking-wider text-gray-400">
              AI Intelligence
            </span>
          </div>
        </div>

        {/* Grouped Sidebar Navigation */}
        <div className="flex-1 space-y-6 overflow-y-auto px-4 py-6">
          {sections.map((sec) => {
            const items = navItems.filter((i) => i.section === sec)
            if (items.length === 0) return null

            return (
              <div key={sec} className="space-y-2">
                <p className="px-3 text-[11px] font-semibold uppercase tracking-wider text-gray-400">
                  {sec}
                </p>
                <div className="space-y-1">
                  {items.map((item) => {
                    const active = pathname === item.to
                    const Icon = item.icon
                    return (
                      <Link
                        key={item.label}
                        // eslint-disable-next-line @typescript-eslint/no-explicit-any
                        to={item.to as any}
                        onClick={() => setMobileNavOpen(false)}
                        className={cn(
                          'group flex items-center justify-between rounded-xl px-3.5 py-2.5 text-xs font-semibold transition-all duration-150',
                          active
                            ? 'bg-gray-900 text-white shadow-sm'
                            : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900',
                        )}
                      >
                        <div className="flex items-center gap-3">
                          <Icon
                            className={cn(
                              'h-4 w-4 transition-colors',
                              active ? 'text-white' : 'text-gray-400 group-hover:text-gray-900',
                            )}
                            aria-hidden="true"
                          />
                          <span>{item.label}</span>
                        </div>
                        {item.badge && (
                          <span
                            className={cn(
                              'flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-bold',
                              active
                                ? 'bg-white text-gray-900'
                                : 'bg-gray-100 text-gray-700 group-hover:bg-gray-200',
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
        </div>

        {/* User Profile Footer Tile */}
        <div className="border-t border-gray-100 p-4">
          <div className="flex items-center gap-3 rounded-xl bg-gray-50 p-2.5">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gray-900 text-xs font-bold text-white">
              {user?.full_name?.charAt(0) ?? 'U'}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-xs font-bold text-gray-900">{user?.full_name}</p>
              <p className="text-[11px] text-gray-500 font-medium">{user ? ROLE_LABELS[user.role] : ''}</p>
            </div>
            <button
              type="button"
              onClick={handleLogout}
              title="Sign Out"
              className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-200/80 hover:text-gray-900 transition-colors"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      </aside>

      {/* Mobile Overlay */}
      {mobileNavOpen && (
        <button
          type="button"
          aria-label="Close navigation"
          className="fixed inset-0 z-30 bg-gray-900/30 backdrop-blur-sm lg:hidden"
          onClick={() => setMobileNavOpen(false)}
        />
      )}

      {/* Main Content Body */}
      <div className="flex min-h-screen flex-1 flex-col overflow-x-hidden">
        {/* Sticky Top Header Bar */}
        <header className="sticky top-0 z-20 flex h-20 items-center justify-between border-b border-gray-200/80 bg-white/90 px-6 backdrop-blur-md">
          {/* Left: Mobile Toggle & Page Greeting Title */}
          <div className="flex items-center gap-4">
            <button
              type="button"
              className="rounded-xl border border-gray-200 p-2 text-gray-600 hover:bg-gray-100 lg:hidden"
              onClick={() => setMobileNavOpen((v) => !v)}
              aria-label="Toggle navigation"
            >
              {mobileNavOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
            <div>
              <h1 className="font-display text-xl font-bold tracking-tight text-gray-900 flex items-center gap-2">
                {title ?? `Hello, ${user?.full_name?.split(' ')[0] ?? 'User'} 👋`}
              </h1>
              <p className="text-xs font-medium text-gray-500">
                {description ?? 'Welcome to the EduGraph AI Management Dashboard.'}
              </p>
            </div>
          </div>

          {/* Right: Search Bar, Notifications & User Avatar Pill */}
          <div className="flex items-center gap-4">
            {/* Search Input Bar matching Inspo */}
            <div className="relative hidden md:block w-64 lg:w-80">
              <Search className="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search anything..."
                className="w-full rounded-xl border border-gray-200/80 bg-gray-50/80 pl-10 pr-4 py-2 text-xs font-medium text-gray-900 placeholder-gray-400 outline-none transition-all focus:border-gray-900 focus:bg-white focus:ring-1 focus:ring-gray-900"
              />
            </div>

            {/* Quick Actions / Page Actions */}
            {actions}

            {/* Message Icon Button */}
            <button
              type="button"
              className="relative hidden sm:flex h-9 w-9 items-center justify-center rounded-xl border border-gray-200/80 bg-gray-50 text-gray-600 hover:bg-gray-100 hover:text-gray-900 transition-colors"
            >
              <MessageSquare className="h-4 w-4" />
            </button>

            {/* Notification Bell Popover */}
            <NotificationBell />

            {/* User Profile Pill Capsule */}
            <div className="hidden sm:flex items-center gap-2.5 rounded-xl border border-gray-200/80 bg-gray-50/80 p-1.5 pr-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gray-900 text-xs font-bold text-white">
                {user?.full_name?.charAt(0) ?? 'A'}
              </div>
              <div className="text-left">
                <p className="text-xs font-bold text-gray-900 leading-tight">
                  {user?.full_name ?? 'User Name'}
                </p>
                <p className="text-[10px] font-medium text-gray-500 leading-tight capitalize">
                  {user ? ROLE_LABELS[user.role] : 'Admin'}
                </p>
              </div>
              <ChevronDown className="h-3.5 w-3.5 text-gray-400" />
            </div>
          </div>
        </header>

        {/* Dynamic Page Content */}
        <main className="flex-1 px-4 py-6 sm:px-6 lg:px-8 space-y-6 max-w-7xl mx-auto w-full">
          {children}
        </main>
      </div>
    </div>
  )
}
