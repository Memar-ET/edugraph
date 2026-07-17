import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'

import { useAuthStore } from '@stores/auth.store'
import type { AuthResponse, Envelope } from '@/types/api'

// Relative by default so the Vite dev-server proxy (vite.config.ts) handles
// routing to the Go API without the browser needing an absolute URL. Set
// VITE_API_URL to override for non-proxied deployments.
export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api/v1',
  headers: { 'Content-Type': 'application/json' },
})

apiClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken
  if (token) {
    config.headers.set('Authorization', `Bearer ${token}`)
  }
  return config
})

type RetriableConfig = InternalAxiosRequestConfig & { _retried?: boolean }

// Only one refresh call in flight at a time -- concurrent 401s from several
// in-flight requests all await the same promise instead of each triggering
// their own refresh (see architecture.docx §9.1.1 "silent refresh").
let refreshPromise: Promise<string | null> | null = null

async function refreshAccessToken(): Promise<string | null> {
  const { refreshToken, setAuth, clearAuth } = useAuthStore.getState()
  if (!refreshToken) return null
  try {
    const res = await axios.post<Envelope<AuthResponse>>(
      `${apiClient.defaults.baseURL}/auth/refresh`,
      { refresh_token: refreshToken },
    )
    const auth = res.data.data
    if (!auth) throw new Error('empty refresh response')
    setAuth(auth)
    return auth.access_token
  } catch {
    clearAuth()
    return null
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
      refreshPromise ??= refreshAccessToken().finally(() => {
        refreshPromise = null
      })
      const newToken = await refreshPromise
      if (newToken) {
        original.headers.set('Authorization', `Bearer ${newToken}`)
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
