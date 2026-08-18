/** Human-friendly percent (max two decimals, trailing zeros dropped — never 8.88888888%). */
export function formatPercent(value: number): string {
  if (!Number.isFinite(value)) return '—'
  const rounded = Math.round(value * 100) / 100
  return `${rounded}%`
}

export function formatMs(value: number): string {
  return `${Math.round(value)} ms`
}

/** Thousands-separated token count, e.g. "1,234". */
export function formatTokens(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0'
  return Math.round(value).toLocaleString()
}

/** Inference speed, e.g. "64.3 tok/s"; a dash when no generation time was recorded. */
export function formatTokensPerSec(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '—'
  return `${(Math.round(value * 10) / 10).toLocaleString()} tok/s`
}

/** Compact relative time, e.g. "2h ago", "3d ago", or a short date for older items. */
export function relativeTime(iso: string | undefined | null): string {
  if (!iso) return ''
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return ''
  const diffMs = Date.now() - then
  const minutes = Math.floor(diffMs / 60000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  return new Date(then).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}
