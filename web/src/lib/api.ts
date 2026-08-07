import axios, {
  AxiosError,
  type AxiosInstance,
  type InternalAxiosRequestConfig,
} from 'axios'
import type { ErrorResponse } from '@/lib/types'

const BASE_URL = '/api/watchtower/v1'

const ACCESS_KEY = 'wt_access_token'
const REFRESH_KEY = 'wt_refresh_token'

export const tokenStore = {
  getAccess: () => localStorage.getItem(ACCESS_KEY),
  getRefresh: () => localStorage.getItem(REFRESH_KEY),
  set: (access: string, refresh: string) => {
    localStorage.setItem(ACCESS_KEY, access)
    localStorage.setItem(REFRESH_KEY, refresh)
  },
  clear: () => {
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
  },
}

export interface LoginPayload {
  username: string
  password: string
}

let api: AxiosInstance

function createApi(): AxiosInstance {
  const instance = axios.create({
    baseURL: BASE_URL,
    headers: { 'Content-Type': 'application/json' },
  })

  instance.interceptors.request.use((config: InternalAxiosRequestConfig) => {
    const token = tokenStore.getAccess()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
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
        const refresh = tokenStore.getRefresh()
        if (refresh) {
          try {
            const { data } = await axios.post<{ data: { access_token: string } }>(
              `${BASE_URL}/auth/refresh-token`,
              { refresh_token: refresh },
            )
            tokenStore.set(data.data.access_token, refresh)
            original.headers.Authorization = `Bearer ${data.data.access_token}`
            return instance(original)
          } catch {
            tokenStore.clear()
            window.location.href = '/login'
          }
        } else {
          tokenStore.clear()
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
