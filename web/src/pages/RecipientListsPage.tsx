import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Loader2, Plus, Pencil, Trash2, Users } from 'lucide-react'
import { getApi, getErrorMessage } from '@/lib/api'
import type { PageInfo, RecipientList } from '@/lib/types'
import { PaginationControls } from '@/components/Pagination'
import { useDebouncedValue } from '@/lib/useDebounce'
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

export function RecipientListsPage() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const debouncedSearch = useDebouncedValue(search)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<RecipientList | null>(null)
  const [deleting, setDeleting] = useState<RecipientList | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['recipient-lists', page, debouncedSearch],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: RecipientList[]; page_info: PageInfo }>(
        '/recipient-lists',
        { params: { page, page_size: PAGE_SIZE, search: debouncedSearch || undefined } },
      )
      return data
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      await getApi().delete(`/recipient-lists/${id}`)
    },
    onSuccess: () => {
      toast.success('Recipient list deleted')
      setDeleting(null)
      queryClient.invalidateQueries({ queryKey: ['recipient-lists'] })
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
          <h1 className="text-2xl font-bold">Recipient Lists</h1>
          <p className="text-sm text-muted-foreground">
            Named groups of email addresses that alert rules can target.
          </p>
        </div>
        <Button
          onClick={() => {
            setEditing(null)
            setDialogOpen(true)
          }}
        >
          <Plus className="mr-2 h-4 w-4" />
          New list
        </Button>
      </div>

      <div className="flex items-center gap-2">
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search by name..."
          className="max-w-xs"
        />
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Emails</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="w-28 text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={5} className="py-10 text-center">
                    <Loader2 className="mx-auto h-6 w-6 animate-spin text-muted-foreground" />
                  </TableCell>
                </TableRow>
              ) : (data?.data ?? []).length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={5}
                    className="py-10 text-center text-muted-foreground"
                  >
                    <Users className="mx-auto mb-2 h-8 w-8" />
                    No recipient lists yet.
                  </TableCell>
                </TableRow>
              ) : (
                (data?.data ?? []).map((list) => (
                  <TableRow key={list.id}>
                    <TableCell className="text-muted-foreground">{list.id}</TableCell>
                    <TableCell className="font-medium">{list.name}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {list.emails.map((email) => (
                          <Badge key={email} variant="secondary">
                            {email}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {new Date(list.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => {
                          setEditing(list)
                          setDialogOpen(true)
                        }}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive"
                        onClick={() => setDeleting(list)}
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

      <ListDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        editing={editing}
        onSaved={() => {
          setDialogOpen(false)
          queryClient.invalidateQueries({ queryKey: ['recipient-lists'] })
        }}
      />

      <AlertDialog open={!!deleting} onOpenChange={(o) => !o && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete recipient list?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently remove &quot;{deleting?.name}&quot;. This action
              cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground"
              onClick={() => deleting && deleteMutation.mutate(deleting.id)}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

interface ListDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  editing: RecipientList | null
  onSaved: () => void
}

function ListDialog({ open, onOpenChange, editing, onSaved }: ListDialogProps) {
  const [name, setName] = useState('')
  const [emails, setEmails] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (open) {
      setName(editing?.name ?? '')
      setEmails(editing?.emails.join(', ') ?? '')
    }
  }, [open, editing])

  const save = async () => {
    setSaving(true)
    try {
      const emailList = emails
        .split(',')
        .map((e) => e.trim())
        .filter(Boolean)
      const payload = { name, emails: emailList }
      if (editing) {
        await getApi().put(`/recipient-lists/${editing.id}`, payload)
        toast.success('Recipient list updated')
      } else {
        await getApi().post('/recipient-lists', payload)
        toast.success('Recipient list created')
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
          <DialogTitle>{editing ? 'Edit list' : 'New recipient list'}</DialogTitle>
          <DialogDescription>
            Emails should be comma-separated.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="list-name">Name</Label>
            <Input
              id="list-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="backend-team"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="list-emails">Emails</Label>
            <Input
              id="list-emails"
              value={emails}
              onChange={(e) => setEmails(e.target.value)}
              placeholder="dev1@example.com, dev2@example.com"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={save}
            disabled={saving || !name.trim() || !emails.trim()}
          >
            {saving && <Loader2 className="h-4 w-4 animate-spin" />}
            {editing ? 'Save changes' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
