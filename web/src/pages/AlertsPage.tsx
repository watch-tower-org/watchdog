import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { History, Loader2 } from 'lucide-react'
import { getApi } from '@/lib/api'
import type { AlertLog, AlertRule, PageInfo } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
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
import { Card, CardContent } from '@/components/ui/card'
import { Link } from 'react-router-dom'

const PAGE_SIZE = 10

export function AlertsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [page, setPage] = useState(1)
  const issueFilter = searchParams.get('issue_id') ?? ''
  const ruleFilter = searchParams.get('rule_id') ?? ''

  const setFilter = (key: string, value: string) => {
    const next = new URLSearchParams(searchParams)
    if (value) next.set(key, value)
    else next.delete(key)
    setSearchParams(next)
    setPage(1)
  }

  const { data, isLoading } = useQuery({
    queryKey: ['alerts', page, issueFilter, ruleFilter],
    queryFn: async () => {
      const params: Record<string, string> = {
        page: String(page),
        page_size: String(PAGE_SIZE),
      }
      if (issueFilter) params.issue_id = issueFilter
      if (ruleFilter) params.rule_id = ruleFilter
      const { data } = await getApi().get<{ data: AlertLog[]; page_info: PageInfo }>(
        '/alerts',
        { params },
      )
      return data
    },
  })

  const formatDate = (iso: string) => new Date(iso).toLocaleString()

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Alert History</h1>
        <p className="text-sm text-muted-foreground">
          Emails WatchTower has sent for alert rules.
        </p>
      </div>

      <div className="flex items-center gap-3">
        {issueFilter && (
          <Badge variant="secondary" className="gap-1">
            issue #{issueFilter}
            <button
              className="ml-1 text-muted-foreground hover:text-foreground"
              onClick={() => setFilter('issue_id', '')}
            >
              ×
            </button>
          </Badge>
        )}
        {ruleFilter && (
          <Badge variant="secondary" className="gap-1">
            rule #{ruleFilter}
            <button
              className="ml-1 text-muted-foreground hover:text-foreground"
              onClick={() => setFilter('rule_id', '')}
            >
              ×
            </button>
          </Badge>
        )}
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">Filter:</span>
          <Select
            value={ruleFilter}
            onValueChange={(v) => setFilter('rule_id', v)}
          >
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="All rules" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">All rules</SelectItem>
              <RuleFilterOptions />
            </SelectContent>
          </Select>
        </div>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Sent at</TableHead>
                <TableHead>Issue</TableHead>
                <TableHead>Rule</TableHead>
                <TableHead>Recipients</TableHead>
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
                    <History className="mx-auto mb-2 h-8 w-8" />
                    No alerts sent yet.
                  </TableCell>
                </TableRow>
              ) : (
                (data?.data ?? []).map((alert) => (
                  <TableRow key={alert.id}>
                    <TableCell className="text-muted-foreground">{alert.id}</TableCell>
                    <TableCell className="whitespace-nowrap">
                      {formatDate(alert.sent_at)}
                    </TableCell>
                    <TableCell>
                      {alert.issue ? (
                        <Link
                          to={`/issues/${alert.issue_id}`}
                          className="font-medium text-primary hover:underline"
                        >
                          {alert.issue.title}
                        </Link>
                      ) : (
                        <span className="text-muted-foreground">
                          #{alert.issue_id}
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      <span className="text-sm">
                        {alert.rule?.name ?? `rule #${alert.rule_id}`}
                      </span>
                    </TableCell>
                    <TableCell className="max-w-xs truncate text-sm text-muted-foreground">
                      {alert.recipients.join(', ')}
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

function RuleFilterOptions() {
  const { data } = useQuery({
    queryKey: ['alert-rules-all'],
    queryFn: async () => {
      const { data } = await getApi().get<{ data: AlertRule[] }>('/alert-rules', {
        params: { page: 1, page_size: 100 },
      })
      return data.data
    },
  })
  return (data ?? []).map((rule) => (
    <SelectItem key={rule.id} value={String(rule.id)}>
      {rule.name}
    </SelectItem>
  ))
}
