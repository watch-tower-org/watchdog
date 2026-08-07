import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { getApi, tokenStore, type LoginPayload } from '@/lib/api'
import type { LoginResponse } from '@/lib/types'

interface AuthContextValue {
  isAuthenticated: boolean
  username: string | null
  login: (payload: LoginPayload) => Promise<LoginResponse>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() =>
    tokenStore.getAccess(),
  )
  const [username, setUsername] = useState<string | null>(null)
  const queryClient = useQueryClient()

  useEffect(() => {
    if (token) {
      const stored = tokenStore.getAccess()
      const parts = stored ? stored.split('.') : []
      if (parts.length === 3) {
        try {
          const payload = JSON.parse(atob(parts[1]))
          setUsername(payload.username || null)
        } catch {
          setUsername(null)
        }
      }
    }
  }, [token])

  const login = async (payload: LoginPayload) => {
    const { data } = await getApi().post<{ data: LoginResponse }>(
      '/auth/login',
      payload,
    )
    tokenStore.set(data.data.access_token, data.data.refresh_token)
    setToken(data.data.access_token)
    setUsername(data.data.username)
    return data.data
  }

  const logout = () => {
    tokenStore.clear()
    setToken(null)
    setUsername(null)
    queryClient.clear()
  }

  return (
    <AuthContext.Provider
      value={{ isAuthenticated: !!token, username, login, logout }}
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
