import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { AlertTriangle, ChevronDown, Loader2, Merge } from 'lucide-react'
import { getApi, getErrorMessage } from '@/lib/api'
import type { Issue, IssueStatus, PageInfo } from '@/lib/types'
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { PaginationControls } from '@/components/Pagination'
import { useDebouncedValue } from '@/lib/useDebounce'
import { cn } from '@/lib/utils'

const PAGE_SIZE = 10

const STATUS_COLORS: Record<IssueStatus, string> = {
  open: 'bg-blue-500/15 text-blue-500',
  resolved: 'bg-emerald-500/15 text-emerald-500',
  muted: 'bg-muted text-muted-foreground',
}

export function statusBadge(status: IssueStatus) {
  return <Badge className={cn(STATUS_COLORS[status])}>{status}</Badge>
}

export function IssuesPage() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [project, setProject] = useState('')
  const [status, setStatus] = useState('')
  const [selected, setSelected] = useState<number[]>([])
  const [mergeOpen, setMergeOpen] = useState(false)
  const [mergeTarget, setMergeTarget] = useState<number | null>(null)

  const debouncedSearch = useDebouncedValue(search)
  const debouncedProject = useDebouncedValue(project)

  const { data, isLoading } = useQuery({
    queryKey: ['issues', page, debouncedSearch, debouncedProject, status],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: Issue[]; page_info: PageInfo }>(
        '/issues',
        {
          params: {
            page,
            page_size: PAGE_SIZE,
            search: debouncedSearch || undefined,
            project: debouncedProject || undefined,
            status: status || undefined,
          },
        },
      )
      return data
    },
  })

  useEffect(() => {
    setPage(1)
  }, [debouncedSearch, debouncedProject, status])

  useEffect(() => {
    setSelected([])
  }, [page, debouncedSearch, debouncedProject, status])

  const pageIds = (data?.data ?? []).map((i) => i.id)
  const allPageSelected = pageIds.length > 0 && pageIds.every((id) => selected.includes(id))

  const toggleSelect = (id: number) => {
    setSelected((sel) =>
      sel.includes(id) ? sel.filter((x) => x !== id) : [...sel, id],
    )
  }

  const toggleSelectPage = () => {
    if (allPageSelected) {
      setSelected((sel) => sel.filter((id) => !pageIds.includes(id)))
    } else {
      setSelected((sel) => [...new Set([...sel, ...pageIds])])
    }
  }

  const selectedIssues = (data?.data ?? []).filter((i) => selected.includes(i.id))
  const openMerge = () => {
    setMergeTarget(null)
    setMergeOpen(true)
  }

  const mergeMutation = useMutation({
    mutationFn: async (targetId: number) => {
      const sourceIds = selected.filter((id) => id !== targetId)
      const { data } = await getApi().post<{ data: Issue }>('/issues/merge', {
        source_ids: sourceIds,
        target_id: targetId,
      })
      return data.data
    },
    onSuccess: () => {
      toast.success('Issues merged')
      setSelected([])
      setMergeOpen(false)
      queryClient.invalidateQueries({ queryKey: ['issues'] })
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Issues</h1>
        <p className="text-sm text-muted-foreground">
          Deduplicated errors grouped by fingerprint.
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search by title..."
          className="max-w-xs"
        />
        <Input
          value={project}
          onChange={(e) => setProject(e.target.value)}
          placeholder="Project..."
          className="max-w-[10rem]"
        />
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-[8.5rem]">
            <SelectValue placeholder="All statuses" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            <SelectItem value="open">Open</SelectItem>
            <SelectItem value="resolved">Resolved</SelectItem>
            <SelectItem value="muted">Muted</SelectItem>
          </SelectContent>
        </Select>
        {selected.length >= 2 && (
          <Button size="sm" onClick={openMerge} className="ml-auto">
            <Merge className="mr-2 h-4 w-4" />
            Merge {selected.length} issues
          </Button>
        )}
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10">
                  <input
                    type="checkbox"
                    aria-label="Select all on page"
                    checked={allPageSelected}
                    onChange={toggleSelectPage}
                    className="h-4 w-4"
                  />
                </TableHead>
                <TableHead>ID</TableHead>
                <TableHead>Title</TableHead>
                <TableHead>Project</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Count</TableHead>
                <TableHead>Last seen</TableHead>
                <TableHead className="w-32">Actions</TableHead>
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
                    <AlertTriangle className="mx-auto mb-2 h-8 w-8" />
                    No issues found.
                  </TableCell>
                </TableRow>
              ) : (
                (data?.data ?? []).map((issue) => (
                  <TableRow key={issue.id}>
                    <TableCell>
                      <input
                        type="checkbox"
                        aria-label={`Select issue ${issue.id}`}
                        checked={selected.includes(issue.id)}
                        onChange={() => toggleSelect(issue.id)}
                        className="h-4 w-4"
                      />
                    </TableCell>
                    <TableCell className="text-muted-foreground">{issue.id}</TableCell>
                    <TableCell className="max-w-md">
                      <Link
                        to={`/issues/${issue.id}`}
                        className="font-medium hover:underline"
                        title={issue.fingerprint}
                      >
                        {issue.title}
                      </Link>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{issue.project || '—'}</Badge>
                    </TableCell>
                    <TableCell>{statusBadge(issue.status)}</TableCell>
                    <TableCell>
                      <span className="font-mono text-sm">{issue.count}</span>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {new Date(issue.last_seen).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <StatusMenu
                        issue={issue}
                        onChanged={() =>
                          queryClient.invalidateQueries({ queryKey: ['issues'] })
                        }
                      />
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

      <Dialog open={mergeOpen} onOpenChange={setMergeOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Merge {selected.length} issues</DialogTitle>
            <DialogDescription>
              All events from the other selected issues move into the target
              issue, which then absorbs their counts and timestamps. The
              sources are deleted.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>Keep as the merged issue</Label>
            <div className="space-y-1.5">
              {selectedIssues.map((issue) => (
                <label
                  key={issue.id}
                  className="flex cursor-pointer items-center gap-2 rounded-md border p-2 text-sm"
                >
                  <input
                    type="radio"
                    name="merge-target"
                    checked={mergeTarget === issue.id}
                    onChange={() => setMergeTarget(issue.id)}
                    className="h-4 w-4"
                  />
                  <span className="min-w-0 flex-1 truncate">{issue.title}</span>
                  <Badge variant="outline">#{issue.id}</Badge>
                </label>
              ))}
              {selected.length > selectedIssues.length && (
                <p className="text-xs text-muted-foreground">
                  {selected.length - selectedIssues.length} more selected on other pages
                  will be merged into the target.
                </p>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setMergeOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => mergeTarget && mergeMutation.mutate(mergeTarget)}
              disabled={!mergeTarget || mergeMutation.isPending}
            >
              {mergeMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              Merge
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function StatusMenu({
  issue,
  onChanged,
}: {
  issue: Issue
  onChanged: () => void
}) {
  const mutation = useMutation({
    mutationFn: async (status: IssueStatus) => {
      await getApi().put(`/issues/${issue.id}`, { status })
    },
    onSuccess: () => {
      toast.success('Issue status updated')
      onChanged()
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  })

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" disabled={mutation.isPending}>
          {mutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <>
              Change
              <ChevronDown className="ml-1 h-3 w-3" />
            </>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => mutation.mutate('open')}>
          Open
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => mutation.mutate('resolved')}>
          Resolve
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => mutation.mutate('muted')}>
          Mute
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
