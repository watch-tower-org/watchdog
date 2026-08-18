import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import {
  Activity,
  AlertTriangle,
  BellRing,
  Bug,
  Hash,
  KeyRound,
  Layers,
  Radar,
  Settings2,
  TrendingUp,
  Users,
} from 'lucide-react'
import { getApi } from '@/lib/api'
import type { AlertSettings, DashboardTrends, EmailSettings, Issue, Settings } from '@/lib/types'
import { timeAgo } from '@/lib/timeAgo'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ChartCard } from '@/components/charts/ChartCard'
import { AreaTrendChart } from '@/components/charts/AreaTrendChart'
import { StatusDonutChart } from '@/components/charts/StatusDonutChart'
import { TopBarChart } from '@/components/charts/TopBarChart'

interface Summary {
  recipient_lists: number
  api_keys: number
  issues: number
  events: number
  alerts: number
}

const RANGES = [
  { value: '24h', label: '24h' },
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
] as const

type RangeKey = (typeof RANGES)[number]['value']

const statusVariant: Record<Issue['status'], 'default' | 'secondary' | 'outline'> = {
  open: 'default',
  resolved: 'secondary',
  muted: 'outline',
}

export function DashboardPage() {
  const [range, setRange] = useState<RangeKey>('7d')

  const { data: summary } = useQuery({
    queryKey: ['dashboard-summary'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: Summary }>('/dashboard/summary')
      return data.data
    },
    refetchInterval: 15000,
    refetchOnWindowFocus: true,
  })

  const { data: trends } = useQuery({
    queryKey: ['dashboard-trends', range],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: DashboardTrends }>('/dashboard/trends', {
        params: { range },
      })
      return data.data
    },
    refetchInterval: 15000,
    refetchOnWindowFocus: true,
  })

  const { data: recent } = useQuery({
    queryKey: ['issues', 'recent'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: Issue[] }>('/issues', {
        params: { page: 1, page_size: 5, status: 'open' },
      })
      return data.data
    },
    refetchInterval: 15000,
  })

  const { data: emailSettings } = useQuery({
    queryKey: ['email-settings'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: EmailSettings }>('/settings/email')
      return data.data
    },
  })

  const { data: alertSettings } = useQuery({
    queryKey: ['alert-settings'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: AlertSettings }>('/settings/alert')
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
    { title: 'Recipient lists', value: summary?.recipient_lists ?? 0, icon: Users, to: '/recipient-lists' },
    { title: 'API keys', value: summary?.api_keys ?? 0, icon: KeyRound, to: '/api-keys' },
    { title: 'Issues', value: summary?.issues ?? 0, icon: AlertTriangle, to: '/issues' },
    { title: 'Events', value: summary?.events ?? 0, icon: Radar, to: '/events' },
    { title: 'Alerts sent', value: summary?.alerts ?? 0, icon: BellRing, to: '/alerts' },
  ]

  const emailConfigured = !!emailSettings?.smtp_host && !!emailSettings?.smtp_from_email

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
        {cards.map((card, index) => (
          <motion.div
            key={card.title}
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.35, delay: index * 0.05, ease: 'easeOut' }}
            whileHover={{ y: -3 }}
          >
            <Link to={card.to}>
              <Card className="transition-colors hover:border-primary/50">
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium text-muted-foreground">
                    {card.title}
                  </CardTitle>
                  <card.icon className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-3xl font-bold tabular-nums">{card.value}</div>
                </CardContent>
              </Card>
            </Link>
          </motion.div>
        ))}
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <ChartCard
          title="Activity"
          description="Events, new issues and alerts over time"
          icon={<Activity className="h-4 w-4" />}
          delay={0.15}
          className="lg:col-span-2"
          action={
            <Tabs value={range} onValueChange={(v) => setRange(v as RangeKey)}>
              <TabsList>
                {RANGES.map((r) => (
                  <TabsTrigger key={r.value} value={r.value}>
                    {r.label}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          }
        >
          {trends ? (
            <AreaTrendChart data={trends.buckets ?? []} range={range} />
          ) : (
            <div className="h-72 w-full animate-pulse rounded-lg bg-muted/50" />
          )}
        </ChartCard>

        <ChartCard
          title="Issues by status"
          description="Where your issues stand right now"
          icon={<Layers className="h-4 w-4" />}
          delay={0.2}
        >
          {trends ? (
            <StatusDonutChart data={trends.issues_by_status ?? []} />
          ) : (
            <div className="h-72 w-full animate-pulse rounded-lg bg-muted/50" />
          )}
        </ChartCard>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <ChartCard
          title="Top projects"
          description="Open issues by project"
          icon={<TrendingUp className="h-4 w-4" />}
          delay={0.25}
        >
          {trends ? (
            <TopBarChart data={trends.top_projects ?? []} color="var(--chart-1)" emptyMessage="No projects yet" />
          ) : (
            <div className="h-48 w-full animate-pulse rounded-lg bg-muted/50" />
          )}
        </ChartCard>

        <ChartCard
          title="Top tags"
          description="Open issues by tag"
          icon={<Hash className="h-4 w-4" />}
          delay={0.3}
        >
          {trends ? (
            <TopBarChart data={trends.top_tags ?? []} color="var(--chart-2)" emptyMessage="No tags yet" />
          ) : (
            <div className="h-48 w-full animate-pulse rounded-lg bg-muted/50" />
          )}
        </ChartCard>

        <ChartCard
          title="Recent issues"
          description="Latest open issues"
          icon={<Bug className="h-4 w-4" />}
          delay={0.35}
        >
          {recent && recent.length > 0 ? (
            <ul className="divide-y">
              {recent.map((issue) => (
                <li key={issue.id}>
                  <Link
                    to={`/issues/${issue.id}`}
                    className="flex items-start gap-3 py-2.5 transition-colors first:pt-0 last:pb-0 hover:bg-muted/40"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">{issue.title}</p>
                      <p className="mt-0.5 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                        <span className="font-mono">{issue.project}</span>
                        {issue.tag && (
                          <span className="rounded bg-muted px-1 py-px font-mono">{issue.tag}</span>
                        )}
                        <span>·</span>
                        <span>{timeAgo(issue.first_seen)}</span>
                      </p>
                    </div>
                    <div className="flex shrink-0 flex-col items-end gap-1.5">
                      <Badge variant={statusVariant[issue.status]}>{issue.status}</Badge>
                      <span className="text-xs text-muted-foreground tabular-nums">
                        {issue.count}×
                      </span>
                    </div>
                  </Link>
                </li>
              ))}
            </ul>
          ) : recent && recent.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">No open issues</p>
          ) : (
            <div className="h-48 w-full animate-pulse rounded-lg bg-muted/50" />
          )}
        </ChartCard>
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
              <span className={emailConfigured ? 'text-emerald-500' : 'text-destructive'}>
                {emailConfigured ? 'Configured' : 'Not configured'}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Host</span>
              <span className="font-mono text-xs">{emailSettings?.smtp_host || '—'}</span>
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
