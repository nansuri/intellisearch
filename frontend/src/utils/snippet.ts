import { SNIPPET_MAX_CHARS } from '../config/app'

/** Cuts a string approximately at the limit, breaking at a word boundary. */
export function clampSnippet(value: string, max = SNIPPET_MAX_CHARS): string {
  const text = value.trim()
  if (text.length <= max) return text
  const cut = text.slice(0, max)
  const space = cut.lastIndexOf(' ')
  const end = space > max * 0.6 ? space : max
  return `${cut.slice(0, end).trimEnd()}…`
}