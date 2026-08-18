// Frontend build-time configuration. Values default to the numbers below and
// can be overridden per deployment via VITE_* environment variables (see
// .env.example). These are baked in at build time, not read at runtime.

/** Max characters of a web-result snippet before it is truncated with a Read more toggle. */
const snippetChars = Number(import.meta.env.VITE_MAX_SNIPPET_CHARS)
export const SNIPPET_MAX_CHARS = Number.isFinite(snippetChars) && snippetChars > 0 ? snippetChars : 180