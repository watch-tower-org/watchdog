import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { Loader2, Radar } from 'lucide-react'
import { getApi } from '@/lib/api'
import type { IssueEvent, PageInfo } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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

const PAGE_SIZE = 20

export function EventsPage() {
  const [searchParams] = useSearchParams()
  const [page, setPage] = useState(1)
  const [project, setProject] = useState('')
  const issueId = searchParams.get('issue_id') || undefined
  const { data, isLoading } = useQuery({
    queryKey: ['events', page, project, issueId],
    queryFn: async () => {
      const { data } = await getApi().get<{
        data: IssueEvent[]
        page_info: PageInfo
      }>('/events', {
        params: {
          page,
          page_size: PAGE_SIZE,
          project: project || undefined,
          issue_id: issueId,
        },
      })
      return data
    },
  })

  useEffect(() => {
    setPage(1)
  }, [project, issueId])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Events</h1>
          <p className="text-sm text-muted-foreground">
            Raw error occurrences reported by SDKs.
          </p>
        </div>
        {issueId && (
          <Button asChild variant="outline" size="sm">
            <Link to="/events">Clear issue filter</Link>
          </Button>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={project}
          onChange={(e) => setProject(e.target.value)}
          placeholder="Project..."
          className="max-w-[10rem]"
        />
        {issueId && (
          <Badge variant="outline">issue #{issueId}</Badge>
        )}
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Message</TableHead>
                <TableHead>Project</TableHead>
                <TableHead>Tag</TableHead>
                <TableHead>Issue</TableHead>
                <TableHead>Timestamp</TableHead>
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
                    <Radar className="mx-auto mb-2 h-8 w-8" />
                    No events found.
                  </TableCell>
                </TableRow>
              ) : (
                (data?.data ?? []).map((event) => (
                  <TableRow key={event.id}>
                    <TableCell className="text-muted-foreground">
                      {event.id}
                    </TableCell>
                    <TableCell className="max-w-md">
                      <span className="block truncate font-medium">
                        {event.message || '—'}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{event.project || '—'}</Badge>
                    </TableCell>
                    <TableCell>
                      {event.tag ? (
                        <Badge variant="secondary">{event.tag}</Badge>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Link
                        to={`/issues/${event.issue_id}`}
                        className="text-muted-foreground hover:underline"
                      >
                        #{event.issue_id}
                      </Link>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {new Date(event.timestamp).toLocaleString()}
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
