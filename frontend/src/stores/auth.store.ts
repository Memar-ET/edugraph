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

// Capability 2A/2B: exam upload, validate, publish.
export function canAccessTeacherDashboard(role: string | undefined): boolean {
  return role === 'teacher' || role === 'school_admin'
}

// Capability 2C Flow 1: exam-taking.
export function canAccessStudentDashboard(role: string | undefined): boolean {
  return role === 'student'
}

// Single source of truth for "where does this role land after login" --
// used by both LoginPage's post-login redirect and the router's
// already-authenticated guards on / and /login. Every role except
// curriculum_officer lands on '/', which the index route's DashboardRouter
// dispatches to a role-specific dashboard component -- curriculum_officer
// lands on its own dashboard (feature 1.2) instead, since '/' has no
// DashboardRouter case for it.
export function landingPathFor(role: string | undefined): string {
  if (role === 'curriculum_officer') return '/curriculum'
  return '/'
}
