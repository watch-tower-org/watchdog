import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'
import type { TrendItem } from '@/lib/types'
import { compactNumber } from '@/lib/format'

interface StatusDonutChartProps {
  data: TrendItem[]
}

const STATUS_LABELS: Record<string, string> = {
  open: 'Open',
  resolved: 'Resolved',
  muted: 'Muted',
}

const STATUS_COLORS: Record<string, string> = {
  open: 'var(--chart-1)',
  resolved: 'var(--chart-2)',
  muted: 'var(--chart-4)',
}

interface DonutTooltipProps {
  active?: boolean
  payload?: { name?: string | number; value?: number | string; payload?: TrendItem }[]
}

function DonutTooltip({ active, payload }: DonutTooltipProps) {
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

/** Donut of issues by status with the total in the center. */
export function StatusDonutChart({ data }: StatusDonutChartProps) {
  const total = data.reduce((sum, item) => sum + item.count, 0)
  const rows = data
    .filter((item) => item.count > 0)
    .map((item) => ({
      name: STATUS_LABELS[item.name] ?? item.name,
      status: item.name,
      count: item.count,
    }))

  return (
    <div className="flex h-full flex-col items-center justify-center gap-4">
      <div className="relative h-52 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Tooltip content={<DonutTooltip />} />
            <Pie
              data={rows}
              dataKey="count"
              nameKey="name"
              innerRadius="62%"
              outerRadius="92%"
              paddingAngle={3}
              cornerRadius={6}
              stroke="transparent"
              animationDuration={800}
              animationBegin={150}
            >
              {rows.map((row) => (
                <Cell key={row.status} fill={STATUS_COLORS[row.status] ?? 'var(--chart-5)'} />
              ))}
            </Pie>
          </PieChart>
        </ResponsiveContainer>
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-2xl font-bold tabular-nums">{compactNumber(total)}</span>
          <span className="text-xs text-muted-foreground">total issues</span>
        </div>
      </div>

      <div className="grid w-full grid-cols-3 gap-2">
        {rows.map((row) => (
          <div
            key={row.status}
            className="flex flex-col items-center gap-1 rounded-lg border bg-muted/40 px-2 py-2"
          >
            <span
              className="inline-block h-2.5 w-2.5 rounded-full"
              style={{ backgroundColor: STATUS_COLORS[row.status] ?? 'var(--chart-5)' }}
            />
            <span className="text-sm font-semibold tabular-nums">{compactNumber(row.count)}</span>
            <span className="text-[11px] leading-none text-muted-foreground">{row.name}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
