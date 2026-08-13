import { useNavigate } from '@tanstack/react-router'

import { Button } from '@components/ui'
import { logout } from '@lib/api/endpoints'
import { useAuthStore } from '@stores/auth.store'

export function AppHeader() {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const clearAuth = useAuthStore((s) => s.clearAuth)

  const handleLogout = () => {
    // Must revoke server-side too -- clearAuth() alone can't touch the
    // HttpOnly session cookies (checklist 11.1), see endpoints.ts's
    // logout() for why.
    void logout()
    clearAuth()
    void navigate({ to: '/login' })
  }

  return (
    <header className="border-b border-gray-200 bg-white">
      <div className="mx-auto flex max-w-4xl items-center justify-between px-4 py-3">
        <span className="text-lg font-semibold text-gray-900">EduGraph AI</span>
        {user && (
          <div className="flex items-center gap-3 text-sm text-gray-600">
            <span>
              {user.full_name} <span className="text-gray-400">({user.role})</span>
            </span>
            <Button variant="ghost" size="sm" onClick={handleLogout}>
              Sign out
            </Button>
          </div>
        )}
      </div>
    </header>
  )
}
