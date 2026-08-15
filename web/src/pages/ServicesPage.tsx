import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  HeartPulse,
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
} from 'lucide-react'
import { getApi, getErrorMessage } from '@/lib/api'
import type {
  CheckNowResponse,
  MonitoredService,
  MonitoredServiceDetail,
  PageInfo,
  RecipientList,
} from '@/lib/types'
import { timeAgo } from '@/lib/timeAgo'
import { formatInterval } from '@/lib/format'
import { PaginationControls } from '@/components/Pagination'
import { useDebouncedValue } from '@/lib/useDebounce'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Card, CardContent } from '@/components/ui/card'

const PAGE_SIZE = 10

const NONE = 'no-emails'

export function ServicesPage() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const debouncedSearch = useDebouncedValue(search)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<MonitoredService | null>(null)
  const [deleting, setDeleting] = useState<MonitoredService | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['monitored-services', page, debouncedSearch],
    queryFn: async () => {
      const { data } = await getApi().get<{
        data: MonitoredService[]
        page_info: PageInfo
      }>('/monitored-services', {
        params: {
          page,
          page_size: PAGE_SIZE,
          search: debouncedSearch || undefined,
        },
      })
      return data
    },
  })

  const checkNowMutation = useMutation({
    mutationFn: async (id: number) => {
      const { data } = await getApi().post<{ data: CheckNowResponse }>(
        `/monitored-services/${id}/check`,
      )
      return data.data
    },
    onSuccess: (res) => {
      const r = res.result
      if (r.status === 'up') {
        toast.success(`UP · ${r.status_code} in ${r.response_time_ms}ms`)
      } else {
        toast.error(`DOWN · ${r.error ?? `HTTP ${r.status_code}`}`)
      }
      queryClient.invalidateQueries({ queryKey: ['monitored-services'] })
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  const toggleMutation = useMutation({
    mutationFn: async ({ id, is_active }: { id: number; is_active: boolean }) => {
      await getApi().put(`/monitored-services/${id}`, { is_active })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['monitored-services'] })
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      await getApi().delete(`/monitored-services/${id}`)
    },
    onSuccess: () => {
      toast.success('Service deleted')
      setDeleting(null)
      queryClient.invalidateQueries({ queryKey: ['monitored-services'] })
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  useEffect(() => {
    setPage(1)
  }, [debouncedSearch])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Services</h1>
          <p className="text-sm text-muted-foreground">
            Monitor the availability of your services by polling a health
            endpoint and get email alerts on state changes.
          </p>
        </div>
        <Button
          onClick={() => {
            setEditing(null)
            setDialogOpen(true)
          }}
        >
          <Plus className="mr-2 h-4 w-4" />
          New service
        </Button>
      </div>

      <div className="flex items-center justify-between">
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search by name or URL…"
          className="w-72"
        />
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Status</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>URL</TableHead>
                <TableHead>Interval</TableHead>
                <TableHead>Last checked</TableHead>
                <TableHead className="w-40 text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={6} className="py-10 text-center">
                    <Loader2 className="mx-auto h-6 w-6 animate-spin text-muted-foreground" />
                  </TableCell>
                </TableRow>
              ) : (data?.data ?? []).length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className="py-10 text-center text-muted-foreground"
                  >
                    <HeartPulse className="mx-auto mb-2 h-8 w-8" />
                    No services yet.
                  </TableCell>
                </TableRow>
              ) : (
                (data?.data ?? []).map((svc) => (
                  <TableRow key={svc.id}>
                    <TableCell>
                      <StatusBadge svc={svc} />
                    </TableCell>
                    <TableCell className="font-medium">
                      <span className={svc.is_active ? '' : 'text-muted-foreground'}>
                        {svc.name}
                      </span>
                      {!svc.is_active && (
                        <Badge variant="outline" className="ml-2">
                          disabled
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="max-w-[240px] truncate font-mono text-xs text-muted-foreground">
                      {svc.url}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatInterval(svc.interval_seconds)}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {svc.last_checked_at ? timeAgo(svc.last_checked_at) : '—'}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="icon"
                        title="Check now"
                        onClick={() => checkNowMutation.mutate(svc.id)}
                        disabled={checkNowMutation.isPending}
                      >
                        <RefreshCw className="h-4 w-4" />
                      </Button>
                      <Switch
                        checked={svc.is_active}
                        onCheckedChange={(checked: boolean) =>
                          toggleMutation.mutate({ id: svc.id, is_active: checked })
                        }
                        disabled={toggleMutation.isPending}
                        className="mx-1 align-middle"
                        aria-label={`Toggle ${svc.name}`}
                      />
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => {
                          setEditing(svc)
                          setDialogOpen(true)
                        }}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive"
                        onClick={() => setDeleting(svc)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <PaginationControls
        pageInfo={data?.page_info}
        page={page}
        onPageChange={setPage}
      />

      <ServiceDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        editing={editing}
        onSaved={() => {
          setDialogOpen(false)
          queryClient.invalidateQueries({ queryKey: ['monitored-services'] })
        }}
      />

      <AlertDialog open={!!deleting} onOpenChange={(o) => !o && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete service?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently remove &quot;{deleting?.name}&quot; and its check
              history.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground"
              onClick={() => deleting && deleteMutation.mutate(deleting.id)}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

function StatusBadge({ svc }: { svc: MonitoredService }) {
  if (svc.last_status === 'up') {
    return <Badge className="bg-emerald-500/15 text-emerald-500">up</Badge>
  }
  if (svc.last_status === 'down') {
    return <Badge className="bg-red-500/15 text-red-500">down</Badge>
  }
  return <Badge variant="outline">never checked</Badge>
}

interface ServiceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  editing: MonitoredService | null
  onSaved: () => void
}

function ServiceDialog({ open, onOpenChange, editing, onSaved }: ServiceDialogProps) {
  const [name, setName] = useState('')
  const [url, setUrl] = useState('')
  const [intervalValue, setIntervalValue] = useState('60')
  const [intervalUnit, setIntervalUnit] = useState<'s' | 'm'>('s')
  const [timeoutSeconds, setTimeoutSeconds] = useState('10')
  const [failuresBeforeAlert, setFailuresBeforeAlert] = useState('1')
  const [recipientListId, setRecipientListId] = useState('')
  const [isActive, setIsActive] = useState(true)
  const [saving, setSaving] = useState(false)

  const { data: lists } = useQuery({
    queryKey: ['recipient-lists-all'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: RecipientList[] }>(
        '/recipient-lists',
        { params: { page: 1, page_size: 100 } },
      )
      return data.data
    },
  })

  const { data: detail } = useQuery({
    queryKey: ['monitored-service', editing?.id],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: MonitoredServiceDetail }>(
        `/monitored-services/${editing!.id}`,
      )
      return data.data
    },
    enabled: open && !!editing,
  })

  useEffect(() => {
    if (open) {
      setName(editing?.name ?? '')
      setUrl(editing?.url ?? '')
      if (editing) {
        const seconds = editing.interval_seconds
        if (seconds % 60 === 0 && seconds >= 60) {
          setIntervalValue(String(seconds / 60))
          setIntervalUnit('m')
        } else {
          setIntervalValue(String(seconds))
          setIntervalUnit('s')
        }
        setTimeoutSeconds(String(editing.timeout_seconds))
        setFailuresBeforeAlert(String(editing.failures_before_alert))
        setRecipientListId(editing.recipient_list_id ? String(editing.recipient_list_id) : NONE)
        setIsActive(editing.is_active)
      } else {
        setIntervalValue('60')
        setIntervalUnit('s')
        setTimeoutSeconds('10')
        setFailuresBeforeAlert('1')
        setRecipientListId(NONE)
        setIsActive(true)
      }
    }
  }, [open, editing])

  const intervalSeconds =
    intervalUnit === 'm'
      ? (Number(intervalValue) || 0) * 60
      : Number(intervalValue) || 0

  const save = async () => {
    setSaving(true)
    try {
      const payload = {
        name,
        url,
        interval_seconds: intervalSeconds,
        timeout_seconds: Number(timeoutSeconds) || 0,
        failures_before_alert: Number(failuresBeforeAlert) || 0,
        recipient_list_id: recipientListId !== NONE ? Number(recipientListId) : null,
        is_active: isActive,
      }
      if (editing) {
        await getApi().put(`/monitored-services/${editing.id}`, payload)
        toast.success('Service updated')
      } else {
        await getApi().post('/monitored-services', payload)
        toast.success('Service created')
      }
      onSaved()
    } catch (err) {
      toast.error(getErrorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{editing ? 'Edit service' : 'New service'}</DialogTitle>
          <DialogDescription>
            WatchTower polls the URL on the configured interval and emails on
            up/down transitions.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="svc-name">Name</Label>
            <Input
              id="svc-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="payments-api"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="svc-url">URL</Label>
            <Input
              id="svc-url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://api.example.com/healthz"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="svc-interval">Check interval</Label>
              <div className="flex gap-2">
                <Input
                  id="svc-interval"
                  type="number"
                  min={1}
                  value={intervalValue}
                  onChange={(e) => setIntervalValue(e.target.value)}
                />
                <Select
                  value={intervalUnit}
                  onValueChange={(v) => setIntervalUnit(v as 's' | 'm')}
                >
                  <SelectTrigger className="w-24">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="s">seconds</SelectItem>
                    <SelectItem value="m">minutes</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="svc-timeout">Timeout (seconds)</Label>
              <Input
                id="svc-timeout"
                type="number"
                min={1}
                value={timeoutSeconds}
                onChange={(e) => setTimeoutSeconds(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Must be less than the check interval.
              </p>
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="svc-failures">Failures before alert</Label>
            <Input
              id="svc-failures"
              type="number"
              min={1}
              value={failuresBeforeAlert}
              onChange={(e) => setFailuresBeforeAlert(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Consecutive failed checks before a DOWN email is sent.
            </p>
          </div>
          <div className="space-y-2">
            <Label>Recipient list</Label>
            <Select value={recipientListId} onValueChange={setRecipientListId}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="No emails (history only)" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NONE}>No emails (history only)</SelectItem>
                {(lists ?? []).map((list) => (
                  <SelectItem key={list.id} value={String(list.id)}>
                    {list.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center gap-2">
            <input
              id="svc-active"
              type="checkbox"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
              className="h-4 w-4"
            />
            <Label htmlFor="svc-active" className="font-normal">
              Service is monitored
            </Label>
          </div>

          {editing && (
            <div className="space-y-2">
              <Label>Recent checks</Label>
              {detail ? (
                detail.recent_checks.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    No checks recorded yet.
                  </p>
                ) : (
                  <div className="max-h-48 space-y-1 overflow-y-auto rounded-md border p-2">
                    {detail.recent_checks.map((check) => (
                      <div
                        key={check.id}
                        className="flex items-center justify-between text-xs"
                      >
                        <span className="text-muted-foreground">
                          {timeAgo(check.checked_at)}
                        </span>
                        <span>
                          {check.status === 'up' ? (
                            <Badge className="bg-emerald-500/15 text-emerald-500">
                              up
                            </Badge>
                          ) : (
                            <Badge className="bg-red-500/15 text-red-500">
                              down
                            </Badge>
                          )}
                        </span>
                        <span className="text-muted-foreground">
                          {check.status_code
                            ? `${check.status_code} · ${check.response_time_ms ?? '?'}ms`
                            : check.error}
                        </span>
                      </div>
                    ))}
                  </div>
                )
              ) : (
                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
              )}
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={save}
            disabled={
              saving ||
              !name.trim() ||
              !url.trim() ||
              intervalSeconds < 10 ||
              Number(timeoutSeconds) >= intervalSeconds
            }
          >
            {saving && <Loader2 className="h-4 w-4 animate-spin" />}
            {editing ? 'Save changes' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}