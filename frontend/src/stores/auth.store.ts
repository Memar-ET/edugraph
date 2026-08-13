import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import type { AuthResponse, UserResponse } from '@/types/api'

interface AuthState {
  // No accessToken/refreshToken here (checklist 11.1, was the exact gap
  // this file's own prior comment already flagged: tokens persisted to
  // localStorage are readable by any script on the page, including an
  // XSS payload). The backend now sets both as HttpOnly cookies -- see
  // backend/pkg/middleware/middleware.go's SetAuthCookies -- which this
  // client never sees the value of at all, only whether a login attempt
  // succeeded. isAuthenticated is that boolean, not a credential: an XSS
  // payload reading `true` out of localStorage gains nothing, it still
  // can't forge a request without the cookie it can't read.
  isAuthenticated: boolean
  user: UserResponse | null
  setAuth: (auth: AuthResponse) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      isAuthenticated: false,
      user: null,
      setAuth: (auth) => set({ isAuthenticated: true, user: auth.user }),
      clearAuth: () => set({ isAuthenticated: false, user: null }),
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
