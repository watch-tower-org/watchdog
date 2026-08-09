import axios, {
  AxiosError,
  type AxiosInstance,
  type InternalAxiosRequestConfig,
} from 'axios'
import type { ErrorResponse } from '@/lib/types'

const BASE_URL = '/api/watchtower/v1'

// Session tokens live in httpOnly cookies set by the server, so they are never
// accessible to JS (XSS-safe) and are sent automatically on same-origin
// requests.

export interface LoginPayload {
  username: string
  password: string
}

let api: AxiosInstance

// The server rotates the refresh token on every refresh call (the presented
// token is revoked and replaced), so parallel 401s must not each fire their own
// refresh: the first rotation would invalidate the token the others are about
// to use. This promise single-flights refreshes — concurrent 401s await the one
// in-flight call, then all retry against the fresh access token.
let refreshPromise: Promise<void> | null = null

function refreshSession(): Promise<void> {
  if (!refreshPromise) {
    // The refresh token travels in the httpOnly cookie automatically.
    refreshPromise = axios
      .post(`${BASE_URL}/auth/refresh-token`, {}, { withCredentials: true })
      .then(() => undefined)
      .finally(() => {
        refreshPromise = null
      })
  }
  return refreshPromise
}

function createApi(): AxiosInstance {
  const instance = axios.create({
    baseURL: BASE_URL,
    withCredentials: true,
    headers: { 'Content-Type': 'application/json' },
  })

  instance.interceptors.response.use(
    (response) => response,
    async (error: AxiosError) => {
      const original = error.config as InternalAxiosRequestConfig & {
        _retry?: boolean
      }

      const isAuthCall =
        original?.url?.includes('/auth/login') ||
        original?.url?.includes('/auth/refresh-token')

      if (
        error.response?.status === 401 &&
        original &&
        !original._retry &&
        !isAuthCall
      ) {
        original._retry = true
        try {
          await refreshSession()
          return instance(original)
        } catch {
          refreshPromise = null
          window.location.href = '/login'
        }
      }

      return Promise.reject(error)
    },
  )

  return instance
}

export function getApi(): AxiosInstance {
  if (!api) {
    api = createApi()
  }
  return api
}

export function getErrorMessage(error: unknown, fallback = 'Something went wrong'): string {
  if (axios.isAxiosError(error)) {
    const err = error.response?.data as ErrorResponse | undefined
    return err?.error || error.message || fallback
  }
  return fallback
}
