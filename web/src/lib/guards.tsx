import { Navigate, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { getApi } from '@/lib/api'
import { useAuth } from '@/lib/auth'
import type { Settings } from '@/lib/types'
import { Loader2 } from 'lucide-react'

export function RequireAuth({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth()
  const location = useLocation()

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return <>{children}</>
}

export function SetupGate({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth()
  const location = useLocation()

  const { data, isLoading } = useQuery({
    queryKey: ['settings'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: Settings }>('/settings')
      return data.data
    },
    retry: false,
  })

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (data && !data.setup_complete && location.pathname !== '/setup') {
    return <Navigate to="/setup" replace />
  }

  return <>{children}</>
}
