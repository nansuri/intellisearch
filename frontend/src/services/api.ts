export type Envelope<T> = { data: T; errorCode: string; errorMessage: string }

export const FRIENDLY_ERRORS: Record<string, string> = {
  AISY01001: 'The AI service is temporarily unavailable — please try again.',
  AISY01002: 'The answer took too long — please try again.',
  AISY01003: 'The AI service returned an error — please try again.',
  AISY02001: "We're busy right now — try again in a moment.",
  AISY02002: "You're asking too quickly — slow down and try again.",
  AISY02003: "You've reached today's question limit — try again tomorrow.",
  AISY02004: 'Guests get one AI search — sign in to continue.',
  AUTH01001: 'Invalid email or password.',
  AUTH01002: 'Your session is invalid or has expired. Please sign in again.',
  AUTH01004: 'Enter your name, a valid email, and a password of at least 8 characters.',
  AUTH01005: 'An account with that email already exists — try signing in instead.',
  AUTH02001: 'You need Super Owner access for that.',
  AISY03002: 'That URL is not allowed — internal or private addresses are blocked.',
  AISY03003: 'That URL is not valid.',
  AISY03004: "We couldn't read that page — it may be unavailable or unreadable.",
  USER02002: 'The image must be a JPG, PNG, GIF, or WebP under 2 MB.',
  USER03001: 'Your search history could not be loaded.',
  USER03002: 'Your search history could not be cleared.',
  NOTE01001: 'Your notes could not be loaded.',
  NOTE01002: 'That note could not be saved.',
  NOTE01003: 'That note no longer exists.',
  NOTE01004: 'That note could not be deleted.',
  TRAN01001: 'The translator is temporarily unavailable — please try again.',
  TRAN01002: 'Enter text to translate and choose a target language.',
  TRAN01003: "You're translating too quickly — slow down and try again.",
  ADMN03002: 'That provider configuration is not valid.',
  ADMN04001: 'That queue configuration is not valid.',
  ADMN05001: 'That site configuration is not valid.',
  ADMN06001: "Couldn't reach the Ollama server — check the base URL and that it's running.",
  ADMN07001: "Couldn't reach the Pollinations account API — try again in a moment.",
  ADMN07002: 'That Pollinations API key was rejected — check it\'s valid and has account access.',
  ADMN07003: 'The image could not be uploaded to Pollinations.',
  ADMN07004: "That Pollinations API key is valid but lacks account access — create one with the account:usage / account:balance scopes on enter.pollinations.ai.",
  ADMN07005: 'Pollinations balance or API-key budget is exhausted — top up or raise the key budget at enter.pollinations.ai.',
  ADMN07006: 'Pollinations is rate-limiting requests — wait a moment and try again.',
  STTS01001: 'Visitor statistics could not be loaded.',
  HTTP_PROXY: 'Backend is not reachable — start the API (run-local.sh) and retry.',
}

export class ApiError extends Error {
  constructor(public code: string, message: string) {
    super(FRIENDLY_ERRORS[code] || message)
  }
}

const base = import.meta.env.VITE_API_BASE_URL || '/api/v1'

// The API always answers with the JSON envelope. If a proxy (nginx/Vite) or a
// stale backend answers with a web page instead (e.g. an SPA fallback or a 502
// error page), parse the envelope safely and surface a clear message rather than
// the cryptic `Unexpected token '<', "<!DOCTYPE ... is not valid JSON"`.
async function parseEnvelope<T>(response: Response, fallbackMessage: string): Promise<Envelope<T>> {
  const type = response.headers.get('Content-Type') || ''
  if (!type.includes('application/json')) {
    throw new ApiError('HTTP_HTML', 'The server returned a web page instead of a JSON response — the API may be outdated or unreachable. Try refreshing.')
  }
  const body = await response.json() as Envelope<T>
  if (!response.ok || body.errorCode) throw new ApiError(body.errorCode || 'HTTP_ERROR', body.errorMessage || fallbackMessage)
  return body
}

export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('token')
  const response = await fetch(`${base}${path}`, { ...options, headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options.headers } })
  return (await parseEnvelope<T>(response, 'The request failed.')).data
}

export async function upload<T>(path: string, field: string, file: File): Promise<T> {
  const token = localStorage.getItem('token')
  const form = new FormData()
  form.append(field, file)
  const response = await fetch(`${base}${path}`, { method: 'POST', headers: token ? { Authorization: `Bearer ${token}` } : {}, body: form })
  return (await parseEnvelope<T>(response, 'The upload failed.')).data
}

export type SiteSettings = { siteName: string; logoUrl: string | null; faviconUrl: string | null; tagline: string | null; copyright: string | null; googleSsoEnabled?: boolean }
export type Source = { position: number; title: string; url: string; domain: string; snippet: string }
export type ImageItem = { position: number; title: string; url: string; thumbnailUrl: string; fullUrl?: string; source?: string; width?: number; height?: number }

export type GeoLocation = { latitude: number; longitude: number; accuracy?: number }
export type AskMode = 'enhanced' | 'search'
export type MapPoint = { label: string; latitude: number; longitude: number }
export type AskResult = { sessionId: string; messageId: string; answer: string; sources: Source[]; images?: ImageItem[]; mapCenter?: MapPoint | null; mapMarkers?: MapPoint[]; visitorId?: string }
export type SessionMessage = { id: string; role: 'system' | 'user' | 'assistant'; content: string; status: string; createdAt: string; sources?: Source[]; images?: ImageItem[]; mapPoints?: MapPoint[] }

// The anonymous guest token: issued by the backend on the first anonymous AI
// ask (also mirrored in an httpOnly cookie). It ties a device to its single
// guest search allowance; the server additionally enforces a per-IP claim, so
// clearing this cannot reset the allowance from the same network.
const VISITOR_KEY = 'visitorId'
export function getVisitorId(): string | null {
  return localStorage.getItem(VISITOR_KEY)
}
function visitorHeaders(): Record<string, string> {
  if (localStorage.getItem('token')) return {}
  const visitor = getVisitorId()
  return visitor ? { 'X-Visitor-ID': visitor } : {}
}
function rememberVisitorId(result: AskResult | undefined) {
  if (result?.visitorId) localStorage.setItem(VISITOR_KEY, result.visitorId)
}
export type ChatSession = { sessionId: string; title: string; createdAt: string; messages: SessionMessage[] }

export type User = { id: string; name: string; email: string; role: 'general_user' | 'super_owner'; status: 'active' | 'suspended'; avatarUrl: string | null; aiDailyQuota: number; lastLoginAt: string | null; createdAt: string }
export type MeResponse = User & { usage: { usedToday: number; quota: number; remaining: number } }
export type LoginResponse = { token: string; user: User }
export type UsersPage = { users: User[]; total: number; page: number; pageSize: number }

export type ProviderType = 'ollama' | 'openai_compatible' | 'pollinations' | 'huggingface'
export type Provider = { id: string; name: string; providerType: ProviderType; baseUrl: string; model: string; parameters: Record<string, unknown> | null; isActive: boolean }
export type HistoryItem = { id: number; query: string; createdAt: string; sessionId?: string; summary?: string }
export type Note = { id: number; userId: string; title: string; content: string; sourceQuery?: string; sessionId?: string; createdAt: string; updatedAt: string }
export type TranslateLanguage = { code: string; name: string }
export type TranslateResult = { translatedText: string }
export type QueueConfig = { maxConcurrent: number; maxQueueSize: number; requestTimeoutMs: number; perUserRateLimit: number; suggestionCacheHours: number; defaultDailyQuota: number; maxImageResults: number }
export type OllamaModel = { name: string; size: number; parameterSize?: string; quantization?: string }
export type OllamaRunningModel = { name: string; size: number; sizeVram: number; cpu?: string; gpu?: string; memory?: string }
export type OllamaHealth = { version: string; runningModels: OllamaRunningModel[] }
export type PollinationsProfile = { githubUsername: string | null; image: string | null; communityEndpointsAllowed: boolean; name?: string | null; email?: string | null }
export type PollinationsKeyInfo = { valid: boolean; type: string; name: string | null; expiresAt: string | null; expiresIn: number | null; permissions: { models: string[] | null; account: string[] | null }; pollenBudget: number | null; rateLimitEnabled: boolean }
export type PollinationsAccount = { ok: boolean; balance: number; profile: PollinationsProfile | null; key: PollinationsKeyInfo | null }
export type PollinationsUsageRecord = { timestamp: string; type: string; model: string | null; api_key: string | null; meter_source: string | null; input_text_tokens: number; output_text_tokens: number; cost_usd: number; response_time_ms: number | null }
export type PollinationsDailyUsage = { date: string; api_key: string | null; model: string | null; meter_source: string | null; requests: number; cost_usd: number }
export type PollinationsModel = { id: string; object?: string; created?: number }
export type PollinationsUploadResult = { id: string; url: string; contentType: string; size: number; tags?: string[] }
export type PollinationsCredentialsInput = { providerId?: string; apiKey?: string; baseUrl?: string }
export type TopQuery = { query: string; count: number }
export type PerUserUsage = { userId: string; name: string; email: string; count: number }
export type UserStats = { questionsToday: number; questionsWeek: number; activeUsersWeek: number; failed: number; registeredUsers: number; anonymousVisitors: number; registerPageVisitors: number; topQueries: TopQuery[]; perUserUsage: PerUserUsage[] }
export type ErrorGroup = { errorCode: string; errorMessage: string; count: number; lastSeen: string }
export type AIStats = { totalCompleted: number; totalFailed: number; successRate: number; totalInputTokens: number; totalOutputTokens: number; tokensPerSec: number; errors: ErrorGroup[]; latency: { averageMs: number; p50: number; p95: number; p99: number }; providers: { providerId: string | null; name: string; model: string; successes: number; total: number; rate: number }[]; queue: { queueDepth: number; inFlight: number; rejected: number; maxConcurrent: number } }
export type TrendPoint = { label: string; count: number }
export type Trends = { daily: TrendPoint[]; weekly: TrendPoint[] }
export type VisitorMetric = { total: number; daily: TrendPoint[]; weekly: TrendPoint[] }
export type VisitorStats = {
  registeredUsers: VisitorMetric
  activeUsers: number
  activeUsers7d: number
  anonymousVisitors: VisitorMetric
  registerPageVisits: VisitorMetric
}
export type WordCount = { word: string; count: number }
export type WordTrendBucket = { label: string; top: WordCount[] }
export type TrendingWords = { window: 'daily' | 'weekly'; buckets: WordTrendBucket[]; overall: WordCount[] }

// Public & ask
export const getSite = () => request<SiteSettings>('/site')
// Best-effort, idempotent: records that an anonymous visitor opened the
// register page so the control panel can measure registration interest. Carries
// the same visitor identity as anonymous AI asks (deduped server-side).
export const trackRegisterVisit = () => request<{ recorded: boolean; new: boolean }>('/stats/register-visit', { method: 'POST', headers: visitorHeaders() })
export const ask = async (query: string, sessionId?: string, location?: GeoLocation, mode: AskMode = 'enhanced') => {
  const body: { query: string; sessionId?: string; location?: GeoLocation; mode?: AskMode } = { query, mode }
  if (sessionId) body.sessionId = sessionId
  if (location) body.location = location
  const result = await request<AskResult>('/ask', { method: 'POST', headers: visitorHeaders(), body: JSON.stringify(body) })
  rememberVisitorId(result)
  return result
}
export const askUrl = async (url: string) => {
  const result = await request<AskResult>('/ask/url', { method: 'POST', headers: visitorHeaders(), body: JSON.stringify({ url }) })
  rememberVisitorId(result)
  return result
}
export const getSession = (id: string) => request<ChatSession>(`/sessions/${id}`)

// Auth & account
export const login = (email: string, password: string) => request<LoginResponse>('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) })
export const register = (name: string, email: string, password: string) => request<LoginResponse>('/auth/register', { method: 'POST', body: JSON.stringify({ name, email, password }) })
export const logout = () => request<{ loggedOut: boolean }>('/auth/logout', { method: 'POST' })
export const getMe = () => request<MeResponse>('/me')
export const updateMe = (name: string, email: string) => request<User>('/me', { method: 'PATCH', body: JSON.stringify({ name, email }) })
export const uploadAvatar = (file: File) => upload<{ avatarUrl: string }>('/me/avatar', 'avatar', file)

// Search history
export const getHistory = (limit = 20) => request<{ items: HistoryItem[] }>(`/me/history?limit=${limit}`)
export const getHistorySuggestions = (refresh = false) => request<{ suggestions: string[] }>(`/me/history/suggestions${refresh ? '?refresh=1' : ''}`)
export const clearHistory = () => request<{ cleared: boolean }>('/me/history', { method: 'DELETE' })

// Mini apps — notes
export const listNotes = () => request<{ items: Note[] }>('/me/notes')
export const createNote = (input: { title: string; content: string; sourceQuery?: string; sessionId?: string }) => request<Note>('/me/notes', { method: 'POST', body: JSON.stringify(input) })
export const updateNote = (id: number, input: { title: string; content: string }) => request<Note>(`/me/notes/${id}`, { method: 'PATCH', body: JSON.stringify(input) })
export const deleteNote = (id: number) => request<{ deleted: boolean }>(`/me/notes/${id}`, { method: 'DELETE' })

// Mini apps — translator (proxied server-side to LibreTranslate)
export const getTranslateLanguages = () => request<{ languages: TranslateLanguage[] }>('/translate/languages')
export const translate = (input: { q: string; source: string; target: string; format?: string }) => request<TranslateResult>('/translate', { method: 'POST', body: JSON.stringify(input) })

// Admin — users
export const listUsers = (q: string, page: number, pageSize = 20) => request<UsersPage>(`/admin/users?q=${encodeURIComponent(q)}&page=${page}&page_size=${pageSize}`)
export const createUser = (input: { name: string; email: string; password: string; role: string; aiDailyQuota: number }) => request<User>('/admin/users', { method: 'POST', body: JSON.stringify(input) })
export const updateUser = (id: string, input: { role?: string; status?: string; aiDailyQuota: number }) => request<User>(`/admin/users/${id}`, { method: 'PATCH', body: JSON.stringify(input) })
export const deleteUser = (id: string) => request<{ deleted: boolean }>(`/admin/users/${id}`, { method: 'DELETE' })

// Admin — statistics
export const getUserStats = () => request<UserStats>('/admin/stats')
export const getTrends = () => request<Trends>('/admin/stats/trends')
export const getTrendingWords = (window: 'daily' | 'weekly' = 'daily') => request<TrendingWords>(`/admin/stats/trending-words?window=${window}`)
export const getVisitorStats = () => request<VisitorStats>('/admin/stats/visitors')
export const getAIStats = (type = '') => request<AIStats>(`/admin/stats/ai${type ? `?type=${encodeURIComponent(type)}` : ''}`)

// Admin — AI providers & queue
export const listProviders = () => request<{ providers: Provider[] }>('/admin/ai/providers')
export const createProvider = (input: { name: string; providerType: string; baseUrl: string; model: string; parameters?: Record<string, unknown> | null; apiKey?: string; isActive: boolean }) => request<Provider>('/admin/ai/providers', { method: 'POST', body: JSON.stringify(input) })
export const updateProvider = (id: string, input: { name?: string; providerType?: string; baseUrl?: string; model?: string; parameters?: Record<string, unknown> | null; apiKey?: string; isActive: boolean }) => request<Provider>(`/admin/ai/providers/${id}`, { method: 'PATCH', body: JSON.stringify(input) })
export const deleteProvider = (id: string) => request<{ deleted: boolean }>(`/admin/ai/providers/${id}`, { method: 'DELETE' })
export const getQueueConfig = () => request<QueueConfig>('/admin/ai/queue-config')
export const updateQueueConfig = (input: QueueConfig) => request<QueueConfig>('/admin/ai/queue-config', { method: 'PATCH', body: JSON.stringify(input) })

// Admin — Ollama introspection (proxied server-side; the browser never calls Ollama)
export const listOllamaModels = (baseUrl: string) => request<{ models: OllamaModel[] }>(`/admin/ai/ollama/models?baseUrl=${encodeURIComponent(baseUrl)}`)
export const getOllamaHealth = (baseUrl: string) => request<OllamaHealth>(`/admin/ai/ollama/health?baseUrl=${encodeURIComponent(baseUrl)}`)

// Admin — Pollinations account introspection (specialized handler, proxied
// server-side; the browser never calls Pollinations). Pass providerId for a
// saved provider (key decrypted server-side) or apiKey+baseUrl for one being
// configured but not yet saved.
export const getPollinationsAccount = (input: PollinationsCredentialsInput) => request<PollinationsAccount>('/admin/ai/pollinations/account', { method: 'POST', body: JSON.stringify(input) })
export const getPollinationsUsage = (input: PollinationsCredentialsInput, days = 30) => request<{ usage: PollinationsUsageRecord[]; count: number }>(`/admin/ai/pollinations/usage?days=${days}`, { method: 'POST', body: JSON.stringify(input) })
export const getPollinationsDailyUsage = (input: PollinationsCredentialsInput, days = 30) => request<{ usage: PollinationsDailyUsage[]; count: number }>(`/admin/ai/pollinations/usage/daily?days=${days}`, { method: 'POST', body: JSON.stringify(input) })
export const getPollinationsModels = (input: PollinationsCredentialsInput) => request<{ models: PollinationsModel[] }>('/admin/ai/pollinations/models', { method: 'POST', body: JSON.stringify(input) })
export const uploadPollinationsImage = async (input: PollinationsCredentialsInput, file: File) => {
  const form = new FormData()
  if (input.providerId) form.append('providerId', input.providerId)
  if (input.apiKey) form.append('apiKey', input.apiKey)
  if (input.baseUrl) form.append('baseUrl', input.baseUrl)
  form.append('file', file)
  const token = localStorage.getItem('token')
  const response = await fetch(`${base}/admin/ai/pollinations/upload`, { method: 'POST', headers: token ? { Authorization: `Bearer ${token}` } : {}, body: form })
  return (await parseEnvelope<PollinationsUploadResult>(response, 'The upload failed.')).data
}

// Admin — branding
export const getSiteSettings = () => request<SiteSettings>('/admin/site-settings')
export const updateSiteSettings = (input: { siteName: string; tagline: string | null; copyright: string | null }) => request<SiteSettings>('/admin/site-settings', { method: 'PATCH', body: JSON.stringify(input) })
export const uploadLogo = (file: File) => upload<{ logoUrl: string }>('/admin/site-settings/logo', 'logo', file)
export const deleteLogo = () => request<{ logoUrl: null }>('/admin/site-settings/logo', { method: 'DELETE' })
export const uploadFavicon = (file: File) => upload<{ faviconUrl: string }>('/admin/site-settings/favicon', 'favicon', file)
export const deleteFavicon = () => request<{ faviconUrl: null }>('/admin/site-settings/favicon', { method: 'DELETE' })