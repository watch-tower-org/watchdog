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
  MoveRight,
} from 'lucide-react'
import { getApi, getErrorMessage } from '@/lib/api'
import type { Issue, IssueEvent, IssueStatus } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import { statusBadge } from '@/pages/IssuesPage'

export function IssueDetailPage() {
  const { id } = useParams<{ id: string }>()
  const queryClient = useQueryClient()
  const [selectedEvents, setSelectedEvents] = useState<number[]>([])
  const [moveOpen, setMoveOpen] = useState(false)
  const [moveMode, setMoveMode] = useState<'existing' | 'new'>('existing')
  const [moveTarget, setMoveTarget] = useState('')
  const [newTitle, setNewTitle] = useState('')
  const [issueSearch, setIssueSearch] = useState('')

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

  const { data: allIssues } = useQuery({
    queryKey: ['issues', 'all-for-move'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: Issue[] }>('/issues', {
        params: { page: 1, page_size: 100 },
      })
      return data.data
    },
  })

  const moveMutation = useMutation({
    mutationFn: async () => {
      const { data } = await getApi().post<{ data: Issue }>('/issues/move-events', {
        event_ids: selectedEvents,
        target_id: moveMode === 'existing' ? Number(moveTarget) : 0,
        title: moveMode === 'new' ? newTitle.trim() || undefined : undefined,
      })
      return data.data
    },
    onSuccess: () => {
      toast.success('Events moved')
      setSelectedEvents([])
      setMoveOpen(false)
      queryClient.invalidateQueries({ queryKey: ['issue', id] })
      queryClient.invalidateQueries({ queryKey: ['issues'] })
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  const openMove = () => {
    setMoveMode('existing')
    setMoveTarget('')
    setNewTitle('')
    setIssueSearch('')
    setMoveOpen(true)
  }

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
  const eligibleIssues = (allIssues ?? []).filter((i) => i.id !== issue.id)
  const filteredIssues = eligibleIssues.filter((i) =>
    issueSearch
      ? i.title.toLowerCase().includes(issueSearch.toLowerCase())
      : true,
  )
  const toggleEvent = (eventId: number) => {
    setSelectedEvents((sel) =>
      sel.includes(eventId)
        ? sel.filter((x) => x !== eventId)
        : [...sel, eventId],
    )
  }

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
          <div className="text-xs text-muted-foreground">
            Fingerprint:{' '}
            <code className="break-all font-mono text-xs">
              {issue.fingerprint}
            </code>
          </div>
        </div>
        <StatusButtons issue={issue} />
      </div>

      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Recent events</h2>
          <div className="flex items-center gap-3">
            {selectedEvents.length > 0 && (
              <Button size="sm" variant="outline" onClick={openMove}>
                <MoveRight className="mr-2 h-4 w-4" />
                Move {selectedEvents.length} event
                {selectedEvents.length === 1 ? '' : 's'}
              </Button>
            )}
            <span className="text-sm text-muted-foreground">
              {events.length} shown
            </span>
          </div>
        </div>
        {events.length === 0 ? (
          <Card>
            <CardContent className="py-10 text-center text-muted-foreground">
              <Bug className="mx-auto mb-2 h-8 w-8" />
              No events recorded for this issue yet.
            </CardContent>
          </Card>
        ) : (
          events.map((event) => (
            <EventCard
              key={event.id}
              event={event}
              selected={selectedEvents.includes(event.id)}
              onToggle={() => toggleEvent(event.id)}
            />
          ))
        )}
      </div>

      <Dialog open={moveOpen} onOpenChange={setMoveOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Move {selectedEvents.length} events</DialogTitle>
            <DialogDescription>
              Reparent the selected events to another issue, or split them into
              a new issue. Counts and timestamps are recomputed automatically.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Destination</Label>
              <Select value={moveMode} onValueChange={(v) => setMoveMode(v as 'existing' | 'new')}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="existing">Existing issue</SelectItem>
                  <SelectItem value="new">New issue (split)</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {moveMode === 'existing' ? (
              <div className="space-y-2">
                <Label htmlFor="move-search">Find issue</Label>
                <Input
                  id="move-search"
                  value={issueSearch}
                  onChange={(e) => setIssueSearch(e.target.value)}
                  placeholder="Search by title..."
                />
                <div className="max-h-48 space-y-1 overflow-y-auto rounded-md border p-2">
                  {filteredIssues.map((candidate) => (
                    <label
                      key={candidate.id}
                      className="flex cursor-pointer items-center gap-2 rounded p-1.5 text-sm hover:bg-muted"
                    >
                      <input
                        type="radio"
                        name="move-target"
                        checked={moveTarget === String(candidate.id)}
                        onChange={() => setMoveTarget(String(candidate.id))}
                        className="h-4 w-4"
                      />
                      <span className="min-w-0 flex-1 truncate">{candidate.title}</span>
                      <Badge variant="outline">#{candidate.id}</Badge>
                    </label>
                  ))}
                  {filteredIssues.length === 0 && (
                    <p className="px-1 py-2 text-xs text-muted-foreground">
                      No matching issues.
                    </p>
                  )}
                </div>
              </div>
            ) : (
              <div className="space-y-2">
                <Label htmlFor="move-title">Title (optional)</Label>
                <Input
                  id="move-title"
                  value={newTitle}
                  onChange={(e) => setNewTitle(e.target.value)}
                  placeholder="Defaults to the first event's message"
                />
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setMoveOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => moveMutation.mutate()}
              disabled={
                moveMutation.isPending ||
                (moveMode === 'existing' && !moveTarget)
              }
            >
              {moveMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              Move
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
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

function EventCard({
  event,
  selected,
  onToggle,
}: {
  event: IssueEvent
  selected: boolean
  onToggle: () => void
}) {
  const [open, setOpen] = useState(false)
  const hasDetails = !!event.stack_trace || !!event.context

  return (
    <Card className={cn(selected && 'border-primary')}>
      <CardHeader className="pb-2">
        <div className="flex w-full items-center gap-2">
          <input
            type="checkbox"
            checked={selected}
            onChange={onToggle}
            onClick={(e) => e.stopPropagation()}
            className="h-4 w-4 shrink-0"
            aria-label="Select event"
          />
          <button
            className="flex min-w-0 flex-1 items-center justify-between gap-2 text-left"
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
        </div>
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
