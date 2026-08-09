import {
  Area,
  AreaChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import type { TrendBucket } from '@/lib/types'
import { compactNumber, formatFull, formatTick } from '@/lib/format'

interface AreaTrendChartProps {
  data: TrendBucket[]
  range: string
}

interface TooltipEntry {
  name?: string
  value?: number | string
  color?: string
  dataKey?: string | number
  stroke?: string
}

interface TrendTooltipProps {
  active?: boolean
  payload?: TooltipEntry[]
  label?: string | number
}

const SERIES: { key: keyof TrendBucket; name: string; color: string; gradient: string }[] = [
  { key: 'events', name: 'Events', color: 'var(--chart-1)', gradient: 'fillEvents' },
  { key: 'new_issues', name: 'New issues', color: 'var(--chart-2)', gradient: 'fillIssues' },
  { key: 'alerts', name: 'Alerts', color: 'var(--chart-3)', gradient: 'fillAlerts' },
]

function TrendTooltip({ active, payload, label, range }: TrendTooltipProps & { range: string }) {
  if (!active || !payload || payload.length === 0) return null
  return (
    <div className="rounded-lg border bg-background/95 px-3 py-2 text-xs shadow-lg backdrop-blur">
      <p className="mb-1.5 font-semibold">{formatFull(String(label ?? ''), range)}</p>
      <div className="space-y-0.5">
        {payload.map((entry) => (
          <p key={String(entry.dataKey)} className="flex items-center gap-1.5">
            <span
              className="inline-block h-2 w-2 rounded-full"
              style={{ backgroundColor: entry.color ?? entry.stroke }}
            />
            <span className="text-muted-foreground">{entry.name}</span>
            <span className="ml-auto pl-3 font-semibold tabular-nums">
              {compactNumber(Number(entry.value ?? 0))}
            </span>
          </p>
        ))}
      </div>
    </div>
  )
}

/** Events / new issues / alerts over the selected range as a stacked-look area chart. */
export function AreaTrendChart({ data, range }: AreaTrendChartProps) {
  const hasData = data.some((b) => b.events > 0 || b.new_issues > 0 || b.alerts > 0)

  return (
    <div className="relative h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <defs>
            {SERIES.map((s) => (
              <linearGradient key={s.gradient} id={s.gradient} x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={s.color} stopOpacity={0.35} />
                <stop offset="95%" stopColor={s.color} stopOpacity={0} />
              </linearGradient>
            ))}
          </defs>
          <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" />
          <XAxis
            dataKey="ts"
            tickFormatter={(v) => formatTick(v, range)}
            tick={{ fontSize: 11 }}
            tickLine={false}
            axisLine={false}
            minTickGap={24}
          />
          <YAxis
            tick={{ fontSize: 11 }}
            tickLine={false}
            axisLine={false}
            width={36}
            allowDecimals={false}
            tickFormatter={(v) => compactNumber(Number(v))}
          />
          <Tooltip content={<TrendTooltip range={range} />} cursor={{ stroke: 'var(--muted-foreground)', strokeDasharray: '4 4' }} />
          <Legend wrapperStyle={{ fontSize: 12 }} iconType="circle" iconSize={8} />
          {SERIES.map((s) => (
            <Area
              key={s.key}
              type="monotone"
              dataKey={s.key}
              name={s.name}
              stroke={s.color}
              strokeWidth={2}
              fill={`url(#${s.gradient})`}
              animationDuration={700}
            />
          ))}
        </AreaChart>
      </ResponsiveContainer>

      {!hasData && (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
          <p className="rounded-full border bg-background/80 px-4 py-1.5 text-sm text-muted-foreground backdrop-blur">
            No activity in this range yet
          </p>
        </div>
      )}
    </div>
  )
}
