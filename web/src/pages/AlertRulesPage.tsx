import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { BellRing, Loader2, Pencil, Plus, Trash2 } from 'lucide-react'
import { getApi, getErrorMessage } from '@/lib/api'
import type { AlertRule, AlertTriggerType, PageInfo, RecipientList } from '@/lib/types'
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
import { Card, CardContent } from '@/components/ui/card'

const PAGE_SIZE = 10

const TRIGGER_LABELS: Record<AlertTriggerType, string> = {
  new_issue: 'New issue',
  spike: 'Spike',
  regression: 'Regression',
}

export function AlertRulesPage() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<AlertRule | null>(null)
  const [deleting, setDeleting] = useState<AlertRule | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['alert-rules', page],
    queryFn: async () => {
      const { data } = await getApi().get<{
        data: AlertRule[]
        page_info: PageInfo
      }>('/alert-rules', { params: { page, page_size: PAGE_SIZE } })
      return data
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      await getApi().delete(`/alert-rules/${id}`)
    },
    onSuccess: () => {
      toast.success('Alert rule deleted')
      setDeleting(null)
      queryClient.invalidateQueries({ queryKey: ['alert-rules'] })
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Alert Rules</h1>
          <p className="text-sm text-muted-foreground">
            Define when WatchTower sends email alerts and to whom.
          </p>
        </div>
        <Button
          onClick={() => {
            setEditing(null)
            setDialogOpen(true)
          }}
        >
          <Plus className="mr-2 h-4 w-4" />
          New rule
        </Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Trigger</TableHead>
                <TableHead>Matcher</TableHead>
                <TableHead>Recipients</TableHead>
                <TableHead>Throttle</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-24 text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={8} className="py-10 text-center">
                    <Loader2 className="mx-auto h-6 w-6 animate-spin text-muted-foreground" />
                  </TableCell>
                </TableRow>
              ) : (data?.data ?? []).length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={8}
                    className="py-10 text-center text-muted-foreground"
                  >
                    <BellRing className="mx-auto mb-2 h-8 w-8" />
                    No alert rules yet.
                  </TableCell>
                </TableRow>
              ) : (
                (data?.data ?? []).map((rule) => (
                  <TableRow key={rule.id}>
                    <TableCell className="text-muted-foreground">{rule.id}</TableCell>
                    <TableCell className="font-medium">{rule.name}</TableCell>
                    <TableCell>
                      <Badge variant="secondary">
                        {TRIGGER_LABELS[rule.trigger_type]}
                      </Badge>
                      {rule.trigger_type === 'spike' && (
                        <span className="ml-2 text-xs text-muted-foreground">
                          ≥{rule.threshold} in {rule.window_minutes}m
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {rule.project ? (
                        <Badge variant="outline">{rule.project}</Badge>
                      ) : (
                        <Badge variant="outline">all projects</Badge>
                      )}
                      {rule.tag && (
                        <Badge variant="secondary" className="ml-1">
                          {rule.tag}
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      {rule.recipient_list?.name ?? `#${rule.recipient_list_id}`}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {rule.throttle_window || 'global'} min
                    </TableCell>
                    <TableCell>
                      {rule.is_active ? (
                        <Badge className="bg-emerald-500/15 text-emerald-500">
                          active
                        </Badge>
                      ) : (
                        <Badge variant="outline">disabled</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => {
                          setEditing(rule)
                          setDialogOpen(true)
                        }}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive"
                        onClick={() => setDeleting(rule)}
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

      <div className="flex items-center justify-between">
        <span className="text-sm text-muted-foreground">
          {data?.page_info.total ?? 0} total
        </span>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={!data?.page_info.has_previous_page}
            onClick={() => setPage((p) => p - 1)}
          >
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {data?.page_info.current_page || 0} / {data?.page_info.total_pages || 0}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={!data?.page_info.has_next_page}
            onClick={() => setPage((p) => p + 1)}
          >
            Next
          </Button>
        </div>
      </div>

      <RuleDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        editing={editing}
        onSaved={() => {
          setDialogOpen(false)
          queryClient.invalidateQueries({ queryKey: ['alert-rules'] })
        }}
      />

      <AlertDialog open={!!deleting} onOpenChange={(o) => !o && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete alert rule?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently remove &quot;{deleting?.name}&quot;. Alert history is
              kept.
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

interface RuleDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  editing: AlertRule | null
  onSaved: () => void
}

function RuleDialog({ open, onOpenChange, editing, onSaved }: RuleDialogProps) {
  const [name, setName] = useState('')
  const [triggerType, setTriggerType] = useState<AlertTriggerType>('new_issue')
  const [project, setProject] = useState('')
  const [tag, setTag] = useState('')
  const [threshold, setThreshold] = useState('10')
  const [windowMinutes, setWindowMinutes] = useState('10')
  const [throttleWindow, setThrottleWindow] = useState('')
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

  useEffect(() => {
    if (open) {
      setName(editing?.name ?? '')
      setTriggerType(editing?.trigger_type ?? 'new_issue')
      setProject(editing?.project ?? '')
      setTag(editing?.tag ?? '')
      setThreshold(editing ? String(editing.threshold || 10) : '10')
      setWindowMinutes(editing ? String(editing.window_minutes || 10) : '10')
      setThrottleWindow(editing && editing.throttle_window ? String(editing.throttle_window) : '')
      setRecipientListId(editing ? String(editing.recipient_list_id) : '')
      setIsActive(editing?.is_active ?? true)
    }
  }, [open, editing])

  const save = async () => {
    setSaving(true)
    try {
      const payload = {
        name,
        trigger_type: triggerType,
        project: project.trim() || undefined,
        tag: tag.trim() || undefined,
        threshold: triggerType === 'spike' ? Number(threshold) || 0 : 0,
        window_minutes: triggerType === 'spike' ? Number(windowMinutes) || 0 : 0,
        throttle_window: throttleWindow ? Number(throttleWindow) : undefined,
        recipient_list_id: Number(recipientListId),
        is_active: isActive,
      }
      if (editing) {
        await getApi().put(`/alert-rules/${editing.id}`, payload)
        toast.success('Alert rule updated')
      } else {
        await getApi().post('/alert-rules', payload)
        toast.success('Alert rule created')
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
          <DialogTitle>{editing ? 'Edit alert rule' : 'New alert rule'}</DialogTitle>
          <DialogDescription>
            Leave matcher fields empty to apply to all projects/tags.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="rule-name">Name</Label>
            <Input
              id="rule-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="backend-team alerts"
            />
          </div>
          <div className="space-y-2">
            <Label>Trigger type</Label>
            <Select value={triggerType} onValueChange={(v) => setTriggerType(v as AlertTriggerType)}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="new_issue">New issue</SelectItem>
                <SelectItem value="spike">Spike (N occurrences in window)</SelectItem>
                <SelectItem value="regression">Regression (resolved reopens)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="rule-project">Project (optional)</Label>
              <Input
                id="rule-project"
                value={project}
                onChange={(e) => setProject(e.target.value)}
                placeholder="all"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="rule-tag">Tag (optional)</Label>
              <Input
                id="rule-tag"
                value={tag}
                onChange={(e) => setTag(e.target.value)}
                placeholder="all"
              />
            </div>
          </div>
          {triggerType === 'spike' && (
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="rule-threshold">Occurrences</Label>
                <Input
                  id="rule-threshold"
                  type="number"
                  min={1}
                  value={threshold}
                  onChange={(e) => setThreshold(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="rule-window">Window (minutes)</Label>
                <Input
                  id="rule-window"
                  type="number"
                  min={1}
                  value={windowMinutes}
                  onChange={(e) => setWindowMinutes(e.target.value)}
                />
              </div>
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="rule-throttle">Throttle window (minutes)</Label>
            <Input
              id="rule-throttle"
              type="number"
              min={1}
              value={throttleWindow}
              onChange={(e) => setThrottleWindow(e.target.value)}
              placeholder="leave empty to use global setting"
            />
          </div>
          <div className="space-y-2">
            <Label>Recipient list</Label>
            <Select value={recipientListId} onValueChange={setRecipientListId}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Select a list" />
              </SelectTrigger>
              <SelectContent>
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
              id="rule-active"
              type="checkbox"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
              className="h-4 w-4"
            />
            <Label htmlFor="rule-active" className="font-normal">
              Rule is active
            </Label>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={save}
            disabled={saving || !name.trim() || !recipientListId}
          >
            {saving && <Loader2 className="h-4 w-4 animate-spin" />}
            {editing ? 'Save changes' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
