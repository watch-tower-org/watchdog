import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { AlertTriangle, ChevronDown, Loader2 } from 'lucide-react'
import { getApi, getErrorMessage } from '@/lib/api'
import type { Issue, IssueStatus, PageInfo } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
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

  const { data, isLoading } = useQuery({
    queryKey: ['issues', page, search, project, status],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: Issue[]; page_info: PageInfo }>(
        '/issues',
        {
          params: {
            page,
            page_size: PAGE_SIZE,
            search: search || undefined,
            project: project || undefined,
            status: status || undefined,
          },
        },
      )
      return data
    },
  })

  useEffect(() => {
    setPage(1)
  }, [search, project, status])

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
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
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
                    <AlertTriangle className="mx-auto mb-2 h-8 w-8" />
                    No issues found.
                  </TableCell>
                </TableRow>
              ) : (
                (data?.data ?? []).map((issue) => (
                  <TableRow key={issue.id}>
                    <TableCell className="text-muted-foreground">{issue.id}</TableCell>
                    <TableCell className="max-w-md">
                      <Link
                        to={`/issues/${issue.id}`}
                        className="font-medium hover:underline"
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
