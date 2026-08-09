import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, BellRing, KeyRound, Radar, Settings2, Users } from 'lucide-react'
import { getApi } from '@/lib/api'
import type { AlertSettings, EmailSettings, Settings } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

interface Summary {
  recipient_lists: number
  api_keys: number
  issues: number
  events: number
  alerts: number
}

export function DashboardPage() {
  const { data: summary } = useQuery({
    queryKey: ['dashboard-summary'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: Summary }>('/dashboard/summary')
      return data.data
    },
    refetchInterval: 15000,
    refetchOnWindowFocus: true,
  })

  const { data: emailSettings } = useQuery({
    queryKey: ['email-settings'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: EmailSettings }>(
        '/settings/email',
      )
      return data.data
    },
  })

  const { data: alertSettings } = useQuery({
    queryKey: ['alert-settings'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: AlertSettings }>(
        '/settings/alert',
      )
      return data.data
    },
  })

  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: Settings }>('/settings')
      return data.data
    },
  })

  const cards = [
    {
      title: 'Recipient lists',
      value: summary?.recipient_lists ?? 0,
      icon: Users,
      to: '/recipient-lists',
    },
    {
      title: 'API keys',
      value: summary?.api_keys ?? 0,
      icon: KeyRound,
      to: '/api-keys',
    },
    {
      title: 'Issues',
      value: summary?.issues ?? 0,
      icon: AlertTriangle,
      to: '/issues',
    },
    {
      title: 'Events',
      value: summary?.events ?? 0,
      icon: Radar,
      to: '/events',
    },
    {
      title: 'Alerts sent',
      value: summary?.alerts ?? 0,
      icon: BellRing,
      to: '/alerts',
    },
  ]

  const emailConfigured =
    !!emailSettings?.smtp_host && !!emailSettings?.smtp_from_email

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="text-sm text-muted-foreground">
          {settings?.setup_complete
            ? 'WatchTower is fully configured.'
            : 'Setup is not yet complete.'}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        {cards.map((card) => (
          <Link key={card.title} to={card.to} className={card.to === '#' ? 'pointer-events-none' : ''}>
            <Card className="transition-colors hover:border-primary/50">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  {card.title}
                </CardTitle>
                <card.icon className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-3xl font-bold">{card.value}</div>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings2 className="h-5 w-5 text-primary" />
              Email settings
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Status</span>
              <span
                className={
                  emailConfigured ? 'text-emerald-500' : 'text-destructive'
                }
              >
                {emailConfigured ? 'Configured' : 'Not configured'}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Host</span>
              <span className="font-mono text-xs">
                {emailSettings?.smtp_host || '—'}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">From</span>
              <span>{emailSettings?.smtp_from_email || '—'}</span>
            </div>
            <div className="pt-2">
              <Button asChild variant="outline" size="sm">
                <Link to="/settings">Manage</Link>
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings2 className="h-5 w-5 text-primary" />
              Alert throttling
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Throttle window</span>
              <span>{alertSettings?.throttle_window ?? 0} min</span>
            </div>
            <div className="pt-2">
              <Button asChild variant="outline" size="sm">
                <Link to="/settings">Manage</Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
