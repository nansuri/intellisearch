export type Envelope<T> = { data: T; errorCode: string; errorMessage: string }

export const FRIENDLY_ERRORS: Record<string, string> = {
  AISY01001: 'The AI service is temporarily unavailable — please try again.',
  AISY01002: 'The answer took too long — please try again.',
  AISY01003: 'The AI service returned an error — please try again.',
  AISY02001: "We're busy right now — try again in a moment.",
  AISY02002: "You're asking too quickly — slow down and try again.",
  AISY02003: "You've reached today's question limit — try again tomorrow.",
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
  ADMN03002: 'That provider configuration is not valid.',
  ADMN04001: 'That queue configuration is not valid.',
  ADMN05001: 'That site configuration is not valid.',
  ADMN06001: "Couldn't reach the Ollama server — check the base URL and that it's running.",
}

export class ApiError extends Error {
  constructor(public code: string, message: string) {
    super(FRIENDLY_ERRORS[code] || message)
  }
}

const base = import.meta.env.VITE_API_BASE_URL || '/api/v1'
export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('token')
  const response = await fetch(`${base}${path}`, { ...options, headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options.headers } })
  const body = await response.json() as Envelope<T>
  if (!response.ok || body.errorCode) throw new ApiError(body.errorCode || 'HTTP_ERROR', body.errorMessage || 'The request failed.')
  return body.data
}

export async function upload<T>(path: string, field: string, file: File): Promise<T> {
  const token = localStorage.getItem('token')
  const form = new FormData()
  form.append(field, file)
  const response = await fetch(`${base}${path}`, { method: 'POST', headers: token ? { Authorization: `Bearer ${token}` } : {}, body: form })
  const body = await response.json() as Envelope<T>
  if (!response.ok || body.errorCode) throw new ApiError(body.errorCode || 'HTTP_ERROR', body.errorMessage || 'The upload failed.')
  return body.data
}

export type SiteSettings = { siteName: string; logoUrl: string | null; faviconUrl: string | null; tagline: string | null; googleSsoEnabled?: boolean }
export type Source = { position: number; title: string; url: string; domain: string; snippet: string }
export type GeoLocation = { latitude: number; longitude: number; accuracy?: number }
export type AskMode = 'enhanced' | 'search'
export type AskResult = { sessionId: string; messageId: string; answer: string; sources: Source[] }
export type SessionMessage = { id: string; role: 'system' | 'user' | 'assistant'; content: string; status: string; createdAt: string; sources?: Source[] }
export type ChatSession = { sessionId: string; title: string; createdAt: string; messages: SessionMessage[] }

export type User = { id: string; name: string; email: string; role: 'general_user' | 'super_owner'; status: 'active' | 'suspended'; avatarUrl: string | null; aiDailyQuota: number; lastLoginAt: string | null; createdAt: string }
export type MeResponse = User & { usage: { usedToday: number; quota: number; remaining: number } }
export type LoginResponse = { token: string; user: User }
export type UsersPage = { users: User[]; total: number; page: number; pageSize: number }

export type ProviderType = 'ollama' | 'openai_compatible' | 'pollinations' | 'huggingface'
export type Provider = { id: string; name: string; providerType: ProviderType; baseUrl: string; model: string; parameters: Record<string, unknown> | null; isActive: boolean }
export type HistoryItem = { id: number; query: string; createdAt: string; sessionId?: string; summary?: string }
export type QueueConfig = { maxConcurrent: number; maxQueueSize: number; requestTimeoutMs: number; perUserRateLimit: number; suggestionCacheHours: number }
export type OllamaModel = { name: string; size: number; parameterSize?: string; quantization?: string }
export type OllamaRunningModel = { name: string; size: number; sizeVram: number; cpu?: string; gpu?: string; memory?: string }
export type OllamaHealth = { version: string; runningModels: OllamaRunningModel[] }
export type TopQuery = { query: string; count: number }
export type PerUserUsage = { userId: string; name: string; email: string; count: number }
export type UserStats = { questionsToday: number; questionsWeek: number; activeUsersWeek: number; failed: number; topQueries: TopQuery[]; perUserUsage: PerUserUsage[] }
export type ErrorGroup = { errorCode: string; errorMessage: string; count: number; lastSeen: string }
export type AIStats = { totalCompleted: number; totalFailed: number; successRate: number; errors: ErrorGroup[]; latency: { averageMs: number; p50: number; p95: number; p99: number }; providers: { providerId: string | null; name: string; model: string; successes: number; total: number; rate: number }[]; queue: { queueDepth: number; inFlight: number; rejected: number; maxConcurrent: number } }
export type TrendPoint = { label: string; count: number }
export type Trends = { daily: TrendPoint[]; weekly: TrendPoint[] }

// Public & ask
export const getSite = () => request<SiteSettings>('/site')
export const ask = (query: string, sessionId?: string, location?: GeoLocation, mode: AskMode = 'enhanced') => {
  const body: { query: string; sessionId?: string; location?: GeoLocation; mode?: AskMode } = { query, mode }
  if (sessionId) body.sessionId = sessionId
  if (location) body.location = location
  return request<AskResult>('/ask', { method: 'POST', body: JSON.stringify(body) })
}
export const askUrl = (url: string) => request<AskResult>('/ask/url', { method: 'POST', body: JSON.stringify({ url }) })
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

// Admin — users
export const listUsers = (q: string, page: number, pageSize = 20) => request<UsersPage>(`/admin/users?q=${encodeURIComponent(q)}&page=${page}&page_size=${pageSize}`)
export const createUser = (input: { name: string; email: string; password: string; role: string; aiDailyQuota: number }) => request<User>('/admin/users', { method: 'POST', body: JSON.stringify(input) })
export const updateUser = (id: string, input: { role?: string; status?: string; aiDailyQuota: number }) => request<User>(`/admin/users/${id}`, { method: 'PATCH', body: JSON.stringify(input) })
export const deleteUser = (id: string) => request<{ deleted: boolean }>(`/admin/users/${id}`, { method: 'DELETE' })

// Admin — statistics
export const getUserStats = () => request<UserStats>('/admin/stats')
export const getTrends = () => request<Trends>('/admin/stats/trends')
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

// Admin — branding
export const getSiteSettings = () => request<SiteSettings>('/admin/site-settings')
export const updateSiteSettings = (input: { siteName: string; tagline: string | null }) => request<SiteSettings>('/admin/site-settings', { method: 'PATCH', body: JSON.stringify(input) })
export const uploadLogo = (file: File) => upload<{ logoUrl: string }>('/admin/site-settings/logo', 'logo', file)
export const deleteLogo = () => request<{ logoUrl: null }>('/admin/site-settings/logo', { method: 'DELETE' })
export const uploadFavicon = (file: File) => upload<{ faviconUrl: string }>('/admin/site-settings/favicon', 'favicon', file)
export const deleteFavicon = () => request<{ faviconUrl: null }>('/admin/site-settings/favicon', { method: 'DELETE' })