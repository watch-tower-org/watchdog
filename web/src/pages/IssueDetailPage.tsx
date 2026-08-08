import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import {
  ArrowLeft,
  Bug,
  ChevronDown,
  ChevronRight,
  Loader2,
} from 'lucide-react'
import { getApi, getErrorMessage } from '@/lib/api'
import type { Issue, IssueEvent, IssueStatus } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import { statusBadge } from '@/pages/IssuesPage'

export function IssueDetailPage() {
  const { id } = useParams<{ id: string }>()

  const { data, isLoading, error } = useQuery({
    queryKey: ['issue', id],
    queryFn: async () => {
      const { data } = await getApi().get<{
        data: { issue: Issue; events: IssueEvent[] }
      }>(`/issues/${id}`)
      return data.data
    },
    enabled: !!id,
  })

  if (isLoading) {
    return (
      <div className="py-20 text-center">
        <Loader2 className="mx-auto h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="py-20 text-center text-muted-foreground">
        Issue not found.
        <div className="mt-4">
          <Button asChild variant="outline">
            <Link to="/issues">Back to issues</Link>
          </Button>
        </div>
      </div>
    )
  }

  const { issue, events } = data

  return (
    <div className="space-y-6">
      <Button asChild variant="ghost" size="sm" className="-ml-2">
        <Link to="/issues">
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to issues
        </Link>
      </Button>

      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0 space-y-2">
          <h1 className="text-2xl font-bold break-words">{issue.title}</h1>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline">{issue.project || '—'}</Badge>
            {issue.tag && <Badge variant="secondary">{issue.tag}</Badge>}
            {statusBadge(issue.status)}
            <span className="text-sm text-muted-foreground">
              {issue.count} occurrence{issue.count === 1 ? '' : 's'}
            </span>
          </div>
          <div className="text-sm text-muted-foreground">
            First seen:{' '}
            {new Date(issue.first_seen).toLocaleString()} · Last seen:{' '}
            {new Date(issue.last_seen).toLocaleString()}
          </div>
        </div>
        <StatusButtons issue={issue} />
      </div>

      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Recent events</h2>
          <span className="text-sm text-muted-foreground">
            {events.length} shown
          </span>
        </div>
        {events.length === 0 ? (
          <Card>
            <CardContent className="py-10 text-center text-muted-foreground">
              <Bug className="mx-auto mb-2 h-8 w-8" />
              No events recorded for this issue yet.
            </CardContent>
          </Card>
        ) : (
          events.map((event) => <EventCard key={event.id} event={event} />)
        )}
      </div>
    </div>
  )
}

function StatusButtons({ issue }: { issue: Issue }) {
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: async (status: IssueStatus) => {
      await getApi().put(`/issues/${issue.id}`, { status })
    },
    onSuccess: () => {
      toast.success('Issue status updated')
      queryClient.invalidateQueries({ queryKey: ['issue', String(issue.id)] })
      queryClient.invalidateQueries({ queryKey: ['issues'] })
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  const options: { value: IssueStatus; label: string }[] = [
    { value: 'open', label: 'Open' },
    { value: 'resolved', label: 'Resolve' },
    { value: 'muted', label: 'Mute' },
  ]

  return (
    <div className="flex gap-2">
      {options.map((opt) => (
        <Button
          key={opt.value}
          variant={issue.status === opt.value ? 'default' : 'outline'}
          size="sm"
          disabled={mutation.isPending}
          onClick={() => mutation.mutate(opt.value)}
        >
          {mutation.isPending && <Loader2 className="mr-1 h-3 w-3 animate-spin" />}
          {opt.label}
        </Button>
      ))}
    </div>
  )
}

function EventCard({ event }: { event: IssueEvent }) {
  const [open, setOpen] = useState(false)
  const hasDetails = !!event.stack_trace || !!event.context

  return (
    <Card>
      <CardHeader className="pb-2">
        <button
          className="flex w-full items-center justify-between gap-2 text-left"
          onClick={() => setOpen((o) => !o)}
          disabled={!hasDetails}
        >
          <div className="flex min-w-0 items-center gap-2">
            {hasDetails &&
              (open ? (
                <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground" />
              ) : (
                <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
              ))}
            <span className="truncate font-medium">{event.message || 'Event'}</span>
          </div>
          <div className="shrink-0 text-xs text-muted-foreground">
            {new Date(event.timestamp).toLocaleString()}
          </div>
        </button>
      </CardHeader>
      {open && (
        <CardContent className="space-y-4 pt-2">
          {event.stack_trace && (
            <div>
              <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                Stack trace
              </div>
              <pre
                className={cn(
                  'overflow-x-auto rounded-md border border-border bg-muted p-3',
                  'font-mono text-xs leading-relaxed',
                )}
              >
                {event.stack_trace}
              </pre>
            </div>
          )}
          {event.context && Object.keys(event.context).length > 0 && (
            <div>
              <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                Context
              </div>
              <pre
                className={cn(
                  'overflow-x-auto rounded-md border border-border bg-muted p-3',
                  'font-mono text-xs leading-relaxed',
                )}
              >
                {JSON.stringify(event.context, null, 2)}
              </pre>
            </div>
          )}
        </CardContent>
      )}
    </Card>
  )
}
