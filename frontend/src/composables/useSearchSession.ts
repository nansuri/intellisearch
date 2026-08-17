import type { ImageItem, MapPoint, Source } from '../services/api'
import type { FollowUpEntry } from '../components/FollowUpBlock.vue'

const STORAGE_KEY = 'search_session_v1'

export type PersistedSearchState = {
  sessionId: string
  query: string
  answer: string
  sources: Source[]
  images: ImageItem[]
  mapCenter?: MapPoint | null
  mapMarkers?: MapPoint[]
  thread: Array<{ question: string; answer: string; sources: Source[]; images: ImageItem[]; mapCenter?: MapPoint | null; mapMarkers?: MapPoint[]; collapsed?: boolean }>
  primaryCollapsed: boolean
  elapsed: number
  savedAt: number
}

export function saveSearchSession(state: PersistedSearchState) {
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch {
    /* quota or private mode */
  }
}

export function loadSearchSession(query: string, sessionId?: string): PersistedSearchState | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as PersistedSearchState
    if (parsed.query !== query) return null
    if (sessionId && parsed.sessionId !== sessionId) return null
    if (!parsed.sessionId || !parsed.answer) return null
    return parsed
  } catch {
    return null
  }
}

export function clearSearchSession() {
  sessionStorage.removeItem(STORAGE_KEY)
}

export function toFollowUpEntries(
  thread: PersistedSearchState['thread'],
  startId = 1,
): FollowUpEntry[] {
  return thread.map((entry, index) => ({
    id: startId + index,
    question: entry.question,
    answer: entry.answer,
    sources: entry.sources || [],
    images: entry.images || [],
    mapCenter: entry.mapCenter ?? null,
    mapMarkers: entry.mapMarkers || [],
    error: null,
    loading: false,
    collapsed: entry.collapsed ?? false,
    highlighted: false,
  }))
}
