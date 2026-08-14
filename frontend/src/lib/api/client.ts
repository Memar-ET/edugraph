import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'

import { useAuthStore } from '@stores/auth.store'
import type { AuthResponse, Envelope } from '@/types/api'

// Relative by default so the Vite dev-server proxy (vite.config.ts) handles
// routing to the Go API without the browser needing an absolute URL. Set
// VITE_API_URL to override for non-proxied deployments.
export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api/v1',
  headers: { 'Content-Type': 'application/json' },
  // checklist 11.1: auth now travels as HttpOnly cookies the backend
  // sets (middleware.SetAuthCookies), not a token this client attaches
  // itself -- withCredentials is what makes axios actually send them.
  // Without this, every request would silently go out unauthenticated
  // instead of erroring, since there's no token variable left to notice
  // is missing.
  withCredentials: true,
})

type RetriableConfig = InternalAxiosRequestConfig & { _retried?: boolean }

// Only one refresh call in flight at a time -- concurrent 401s from several
// in-flight requests all await the same promise instead of each triggering
// their own refresh (see architecture.docx §9.1.1 "silent refresh").
let refreshPromise: Promise<boolean> | null = null

// No token in, no token out -- the refresh cookie goes out automatically
// (withCredentials), and a successful response's Set-Cookie headers
// update the access-token cookie the same way login did. This function's
// only job is to report whether the browser's cookie jar now has a valid
// session, so the caller knows whether to retry the original request.
async function refreshSession(): Promise<boolean> {
  const { setAuth, clearAuth } = useAuthStore.getState()
  try {
    const res = await axios.post<Envelope<AuthResponse>>(
      `${apiClient.defaults.baseURL}/auth/refresh`,
      undefined,
      { withCredentials: true },
    )
    const auth = res.data.data
    if (!auth) throw new Error('empty refresh response')
    setAuth(auth)
    return true
  } catch {
    clearAuth()
    return false
  }
}

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const original = error.config as RetriableConfig | undefined
    const status = error.response?.status
    const isAuthEndpoint = original?.url?.includes('/auth/login') || original?.url?.includes('/auth/refresh')

    if (status === 401 && original && !original._retried && !isAuthEndpoint) {
      original._retried = true
      refreshPromise ??= refreshSession().finally(() => {
        refreshPromise = null
      })
      const refreshed = await refreshPromise
      if (refreshed) {
        // No header to reattach -- the retried request picks up the
        // freshly-set cookie automatically.
        return apiClient(original)
      }
    }
    return Promise.reject(error)
  },
)

/** Extracts a human-readable message from a failed apiClient call. */
export function apiErrorMessage(err: unknown, fallback = 'Something went wrong. Please try again.'): string {
  if (axios.isAxiosError(err)) {
    const envelope = err.response?.data as Envelope<unknown> | undefined
    if (envelope?.error) return envelope.error
    if (err.message) return err.message
  }
  return fallback
}
