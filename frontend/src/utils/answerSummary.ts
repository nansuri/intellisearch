const MARKDOWN_STRIP = /[#*`_\[\]()!>|~-]/g

/** Plain-text preview for collapsed answer cards. */
export function answerPreview(markdown: string, max = 160): string {
  const plain = markdown.replace(MARKDOWN_STRIP, ' ').replace(/\s+/g, ' ').trim()
  if (plain.length <= max) return plain
  return `${plain.slice(0, max).trim()}…`
}

/** Whether an answer is long enough to auto-collapse when a follow-up starts. */
export function isLongAnswer(text: string): boolean {
  return text.replace(MARKDOWN_STRIP, ' ').trim().length > 180
}
