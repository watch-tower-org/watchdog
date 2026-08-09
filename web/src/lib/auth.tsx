import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { getApi, type LoginPayload } from '@/lib/api'
import type { LoginResponse, MeResponse } from '@/lib/types'

interface AuthContextValue {
  isAuthenticated: boolean
  // checking is true until the initial /auth/me round-trip finishes, so the
  // app doesn't flash to /login on a hard reload.
  checking: boolean
  username: string | null
  login: (payload: LoginPayload) => Promise<LoginResponse>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [username, setUsername] = useState<string | null>(null)
  const [checking, setChecking] = useState(true)
  const queryClient = useQueryClient()

  useEffect(() => {
    let cancelled = false
    getApi()
      .get<{ data: MeResponse }>('/auth/me')
      .then(({ data }) => {
        if (!cancelled) setUsername(data.data.username)
      })
      .catch(() => {
        if (!cancelled) setUsername(null)
      })
      .finally(() => {
        if (!cancelled) setChecking(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const login = async (payload: LoginPayload) => {
    const { data } = await getApi().post<{ data: LoginResponse }>(
      '/auth/login',
      payload,
    )
    setUsername(data.data.username)
    return data.data
  }

  const logout = () => {
    // Best-effort: clear the server-side cookies; local state clears either way.
    getApi().post('/auth/logout').catch(() => undefined)
    setUsername(null)
    queryClient.clear()
  }

  return (
    <AuthContext.Provider
      value={{ isAuthenticated: username !== null, checking, username, login, logout }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return ctx
}
