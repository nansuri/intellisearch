/** Human-friendly percent (max two decimals, trailing zeros dropped — never 8.88888888%). */
export function formatPercent(value: number): string {
  if (!Number.isFinite(value)) return '—'
  const rounded = Math.round(value * 100) / 100
  return `${rounded}%`
}

export function formatMs(value: number): string {
  return `${Math.round(value)} ms`
}
