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

/**
 * Humanizes a check interval in seconds, e.g. 30 -> "30s", 300 -> "5m",
 * 3600 -> "1h", 86400 -> "1d".
 */
export function formatInterval(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 60) {
    return `${Math.max(seconds, 0)}s`
  }
  if (seconds < 3600) {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return s ? `${m}m ${s}s` : `${m}m`
  }
  if (seconds < 86400) {
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    return m ? `${h}h ${m}m` : `${h}h`
  }
  const d = Math.floor(seconds / 86400)
  return `${d}d`
}
