// useMiniAppRunner powers every mini-app iframe in the app — the Studio's live
// preview and the public runner share it, so the execution boundary is identical
// everywhere.
//
// Security model: the app runs in a SAME-ORIGIN sandboxed iframe (scripts +
// same-origin + forms + popups allowed, but no top-navigation). Because it is
// same-origin it can call the platform API with the signed-in user's token; a
// <base target="_blank"> forces any link/form to open a new tab instead of
// navigating the shell away. No external network access is needed or granted
// beyond what the platform API offers.

export interface MiniAppSource {
  html: string
  css: string
  js: string
}

// The sandbox flags we permit. Deliberately excludes allow-top-navigation and
// allow-pointer-lock. allow-same-origin + allow-scripts together give the app
// its API access — that is the point of a same-origin runner.
export const MINI_APP_SANDBOX =
  'allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox allow-modals allow-downloads'

export function buildMiniAppDocument({ html, css, js }: MiniAppSource): string {
  return ['<!doctype html>', '<html>', '<head>', '<meta charset="utf-8">', '<base target="_blank">', '<style>', '', css, '</style>', '</head>', '<body>', html, '<script>', js, '<\/script>', '</body>', '</html>'].join('\n')
}

const SAMPLE: Record<'html' | 'css' | 'js', string> = {
  html: '<h1>Hello, mini app</h1>\n<p>Try editing the HTML, CSS, or JS — the preview updates live.</p>',
  css: 'h1 { color: var(--color-primary, #4f6ef7); }\np { color: var(--color-muted, #666); }',
  js: `// Runs inside a sandboxed, same-origin iframe.
// async function api(path, options = {}) { ... } is injected for you.
// Uncomment to call the platform AI from your mini app:
// const { data } = await api('/ask', { method: 'POST', body: JSON.stringify({ query: 'hello', mode: 'search' }) })
// console.log(data)`,
}

// snippetFor returns the built-in starter snippet when a field is blank, so the
// "Insert sample" buttons give a brand-new app something to run.
export function snippetFor(part: 'html' | 'css' | 'js', current: string): string {
  if (current.trim()) return current
  return SAMPLE[part]
}

// The full document handed to <iframe srcdoc>. The app's own JS cannot escape
// the frame (sandbox + base target), and the owning page never mixes the app's
// scripts into its own scope.
export function buildDocument(source: MiniAppSource): string {
  return buildMiniAppDocument(source)
}