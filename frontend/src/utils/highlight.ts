// Snippet highlighting for the Google-style web-result list. Terms are derived
// from the user's query (mirroring the backend tokenizer: lowercase, alphanumeric
// only, stopwords and <3-rune words dropped) and wrapped in <mark> tags. The
// snippet text is HTML-escaped before highlighting, so external search snippets
// can never inject markup (XSS-safe).

const STOPWORDS = new Set([
  'the', 'and', 'for', 'are', 'was', 'were', 'with', 'from', 'what', 'when',
  'where', 'which', 'that', 'this', 'your', 'you', 'how', 'why', 'who', 'can',
  'does', 'did', 'not', 'but', 'all', 'any', 'has', 'have', 'its', 'our',
  'about', 'into', 'than', 'then', 'them', 'they', 'will', 'would', 'could',
  'should', 'please', 'tell', 'give', 'find', 'show', 'need', 'want',
])

export function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, (ch) => {
    switch (ch) {
      case '&': return '&amp;'
      case '<': return '&lt;'
      case '>': return '&gt;'
      case '"': return '&quot;'
      default: return '&#39;'
    }
  })
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/** Returns the significant lowercase terms of a query (deduplicated). */
export function significantTerms(query: string): string[] {
  const fields = query.toLowerCase().split(/[^a-z0-9]+/).filter(Boolean)
  const seen = new Set<string>()
  const terms: string[] = []
  for (const field of fields) {
    if (field.length < 3 || STOPWORDS.has(field) || seen.has(field)) continue
    seen.add(field)
    terms.push(field)
  }
  return terms
}

/** Escapes `text` and wraps occurrences of the query's significant terms in <mark>. */
export function highlightTerms(text: string, query: string): string {
  const escaped = escapeHtml(text)
  const terms = significantTerms(query)
  if (!terms.length) return escaped
  const pattern = new RegExp(`(${terms.map(escapeRegExp).join('|')})`, 'gi')
  return escaped.replace(pattern, '<mark>$1</mark>')
}
