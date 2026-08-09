import {
  Bar,
  BarChart,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import type { TrendItem } from '@/lib/types'
import { compactNumber } from '@/lib/format'

interface TopBarChartProps {
  data: TrendItem[]
  color?: string
  emptyMessage?: string
}

interface BarTooltipProps {
  active?: boolean
  payload?: { name?: string | number; value?: number | string; payload?: TrendItem }[]
}

function BarTooltip({ active, payload }: BarTooltipProps) {
  if (!active || !payload || payload.length === 0) return null
  const entry = payload[0]
  return (
    <div className="rounded-lg border bg-background/95 px-3 py-2 text-xs shadow-lg backdrop-blur">
      <p className="font-semibold">{String(entry.name ?? 'Unknown')}</p>
      <p className="text-muted-foreground">
        <span className="font-semibold text-foreground tabular-nums">
          {compactNumber(Number(entry.value ?? 0))}
        </span>{' '}
        issues
      </p>
    </div>
  )
}

/** Horizontal ranked bar list (top projects / top tags). */
export function TopBarChart({ data, color = 'var(--chart-1)', emptyMessage = 'No data yet' }: TopBarChartProps) {
  const rows = data.map((item) => ({
    name: item.name || '—',
    count: item.count,
  }))

  if (rows.length === 0) {
    return (
      <div className="flex h-48 items-center justify-center">
        <p className="text-sm text-muted-foreground">{emptyMessage}</p>
      </div>
    )
  }

  return (
    <div className="h-48 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={rows} layout="vertical" margin={{ top: 0, right: 12, left: 0, bottom: 0 }}>
          <XAxis type="number" hide allowDecimals={false} />
          <YAxis
            type="category"
            dataKey="name"
            width={96}
            tick={{ fontSize: 11 }}
            tickLine={false}
            axisLine={false}
          />
          <Tooltip content={<BarTooltip />} cursor={{ fill: 'var(--muted)', opacity: 0.35 }} />
          <Bar dataKey="count" radius={[0, 4, 4, 0]} barSize={16} animationDuration={700}>
            {rows.map((row) => (
              <Cell key={row.name} fill={color} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
