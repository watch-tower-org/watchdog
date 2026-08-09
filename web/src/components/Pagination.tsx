import { Button } from '@/components/ui/button'
import type { PageInfo } from '@/lib/types'

interface PaginationControlsProps {
  pageInfo?: PageInfo
  page: number
  onPageChange: (page: number) => void
}

export function PaginationControls({
  pageInfo,
  page,
  onPageChange,
}: PaginationControlsProps) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-muted-foreground">
        {pageInfo?.total ?? 0} total
      </span>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={!pageInfo?.has_previous_page}
          onClick={() => onPageChange(page - 1)}
        >
          Previous
        </Button>
        <span className="text-sm text-muted-foreground">
          Page {pageInfo?.current_page || 0} / {pageInfo?.total_pages || 0}
        </span>
        <Button
          variant="outline"
          size="sm"
          disabled={!pageInfo?.has_next_page}
          onClick={() => onPageChange(page + 1)}
        >
          Next
        </Button>
      </div>
    </div>
  )
}
