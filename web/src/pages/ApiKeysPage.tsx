import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  Copy,
  Check,
  KeyRound,
  Loader2,
  Pencil,
  Plus,
  ShieldX,
} from 'lucide-react'
import { getApi, getErrorMessage } from '@/lib/api'
import type { ApiKey, CreateApiKeyResponse, PageInfo } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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

export function ApiKeysPage() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [createOpen, setCreateOpen] = useState(false)
  const [revoking, setRevoking] = useState<ApiKey | null>(null)
  const [editing, setEditing] = useState<ApiKey | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['api-keys', page],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: ApiKey[]; page_info: PageInfo }>(
        '/api-keys',
        { params: { page, page_size: PAGE_SIZE } },
      )
      return data
    },
  })

  const revokeMutation = useMutation({
    mutationFn: async (id: number) => {
      await getApi().put(`/api-keys/${id}/revoke`)
    },
    onSuccess: () => {
      toast.success('API key revoked')
      setRevoking(null)
      queryClient.invalidateQueries({ queryKey: ['api-keys'] })
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">API Keys</h1>
          <p className="text-sm text-muted-foreground">
            Keys used by SDKs to report events.
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create key
        </Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Key</TableHead>
                <TableHead>Project</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Last used</TableHead>
                <TableHead className="w-24 text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={7} className="py-10 text-center">
                    <Loader2 className="mx-auto h-6 w-6 animate-spin text-muted-foreground" />
                  </TableCell>
                </TableRow>
              ) : (data?.data ?? []).length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={7}
                    className="py-10 text-center text-muted-foreground"
                  >
                    <KeyRound className="mx-auto mb-2 h-8 w-8" />
                    No API keys yet.
                  </TableCell>
                </TableRow>
              ) : (
                (data?.data ?? []).map((key) => (
                  <TableRow key={key.id}>
                    <TableCell className="text-muted-foreground">{key.id}</TableCell>
                    <TableCell className="font-medium">{key.name}</TableCell>
                    <TableCell>
                      <code className="font-mono text-xs text-muted-foreground">
                        {key.masked}
                      </code>
                    </TableCell>
                    <TableCell>
                      {key.project ? (
                        <Badge>{key.project}</Badge>
                      ) : (
                        <Badge variant="outline">global</Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      {key.is_active ? (
                        <Badge className="bg-emerald-500/15 text-emerald-500">
                          active
                        </Badge>
                      ) : (
                        <Badge variant="destructive">revoked</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {key.last_used_at
                        ? new Date(key.last_used_at).toLocaleString()
                        : 'never'}
                    </TableCell>
                    <TableCell className="text-right">
                      {key.is_active && (
                        <>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setEditing(key)}
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="text-destructive hover:text-destructive"
                            onClick={() => setRevoking(key)}
                          >
                            <ShieldX className="h-4 w-4" />
                          </Button>
                        </>
                      )}
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

      <CreateKeyDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={() => queryClient.invalidateQueries({ queryKey: ['api-keys'] })}
      />

      <EditKeyDialog
        key={editing?.id ?? 'none'}
        apiKey={editing}
        onOpenChange={(o) => !o && setEditing(null)}
        onSaved={() => {
          setEditing(null)
          queryClient.invalidateQueries({ queryKey: ['api-keys'] })
        }}
      />

      <AlertDialog open={!!revoking} onOpenChange={(o) => !o && setRevoking(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revoke API key?</AlertDialogTitle>
            <AlertDialogDescription>
              SDKs using &quot;{revoking?.name}&quot; will immediately stop being able
              to report events. This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground"
              onClick={() => revoking && revokeMutation.mutate(revoking.id)}
              disabled={revokeMutation.isPending}
            >
              {revokeMutation.isPending && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              Revoke
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

function CreateKeyDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: () => void
}) {
  const [name, setName] = useState('')
  const [project, setProject] = useState('')
  const [saving, setSaving] = useState(false)
  const [result, setResult] = useState<CreateApiKeyResponse | null>(null)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (open) {
      setName('')
      setProject('')
      setResult(null)
      setCopied(false)
    }
  }, [open])

  const create = async () => {
    setSaving(true)
    try {
      const { data } = await getApi().post<{ data: CreateApiKeyResponse }>(
        '/api-keys',
        { name, project: project.trim() },
      )
      setResult(data.data)
      onCreated()
      toast.success('API key created')
    } catch (err) {
      toast.error(getErrorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  const copy = async () => {
    if (!result) return
    await navigator.clipboard.writeText(result.key)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create API key</DialogTitle>
          <DialogDescription>
            {result
              ? 'Copy this key now — you will not see it again.'
              : 'Leave project empty for a global key.'}
          </DialogDescription>
        </DialogHeader>

        {result ? (
          <div className="space-y-4">
            <div className="rounded-md border border-border bg-muted p-3">
              <code className="block break-all font-mono text-sm">{result.key}</code>
            </div>
            <Button className="w-full" onClick={copy}>
              {copied ? <Check className="mr-2 h-4 w-4" /> : <Copy className="mr-2 h-4 w-4" />}
              {copied ? 'Copied!' : 'Copy key'}
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="key-name">Name</Label>
              <Input
                id="key-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="backend-api"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="key-project">Project (optional)</Label>
              <Input
                id="key-project"
                value={project}
                onChange={(e) => setProject(e.target.value)}
                placeholder="leave empty for global"
              />
            </div>
          </div>
        )}

        <DialogFooter>
          {result ? (
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Done
            </Button>
          ) : (
            <>
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button onClick={create} disabled={saving || !name.trim()}>
                {saving && <Loader2 className="h-4 w-4 animate-spin" />}
                Create
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function EditKeyDialog({
  apiKey,
  onOpenChange,
  onSaved,
}: {
  apiKey: ApiKey | null
  onOpenChange: (open: boolean) => void
  onSaved: () => void
}) {
  const [name, setName] = useState('')
  const [project, setProject] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (apiKey) {
      setName(apiKey.name)
      setProject(apiKey.project)
    }
  }, [apiKey])

  const save = async () => {
    if (!apiKey) return
    setSaving(true)
    try {
      await getApi().put(`/api-keys/${apiKey.id}`, {
        name,
        project: project.trim(),
      })
      toast.success('API key updated')
      onSaved()
    } catch (err) {
      toast.error(getErrorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={!!apiKey} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit API key</DialogTitle>
          <DialogDescription>
            Update the name or project label. The key value never changes.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-key-name">Name</Label>
            <Input
              id="edit-key-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="backend-api"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="edit-key-project">Project (optional)</Label>
            <Input
              id="edit-key-project"
              value={project}
              onChange={(e) => setProject(e.target.value)}
              placeholder="leave empty for global"
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={save} disabled={saving || !name.trim()}>
            {saving && <Loader2 className="h-4 w-4 animate-spin" />}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
