import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import type { AuthResponse, UserResponse } from '@/types/api'

interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  user: UserResponse | null
  setAuth: (auth: AuthResponse) => void
  clearAuth: () => void
}

// Tokens are persisted to localStorage for this Phase 1 build (simplest
// path to "survives a page reload"). A production hardening pass should
// move the refresh token to an httpOnly cookie set by the Go API to
// reduce XSS exposure -- tracked as follow-up, not blocking Phase 1.
export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      refreshToken: null,
      user: null,
      setAuth: (auth) =>
        set({
          accessToken: auth.access_token,
          refreshToken: auth.refresh_token,
          user: auth.user,
        }),
      clearAuth: () => set({ accessToken: null, refreshToken: null, user: null }),
    }),
    { name: 'edugraph-auth' },
  ),
)

export function canAccessCurriculumReview(role: string | undefined): boolean {
  return role === 'curriculum_officer' || role === 'ministry_admin'
}
