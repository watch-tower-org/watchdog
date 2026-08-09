/** Compacts large numbers for chart axes/labels, e.g. 1200 -> "1.2k". */
export function compactNumber(value: number): string {
  return new Intl.NumberFormat('en-US', {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value)
}

/** Short tick label for a trend bucket, adapted to the active range. */
export function formatTick(ts: string, range: string): string {
  const d = new Date(ts)
  if (range === '24h') {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

/** Full label for tooltips. */
export function formatFull(ts: string, range: string): string {
  const d = new Date(ts)
  if (range === '24h') {
    return d.toLocaleString([], {
      weekday: 'short',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }
  return d.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' })
}
