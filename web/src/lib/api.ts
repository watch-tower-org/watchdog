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
          // The refresh token travels in the httpOnly cookie automatically.
          await axios.post(
            `${BASE_URL}/auth/refresh-token`,
            {},
            { withCredentials: true },
          )
          return instance(original)
        } catch {
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
