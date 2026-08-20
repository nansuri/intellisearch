<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AskBox from '../components/AskBox.vue'
import AppHeader from '../components/AppHeader.vue'
import ErrorBanner from '../components/ErrorBanner.vue'
import UrlAskBox from '../components/UrlAskBox.vue'
import CollapsibleAnswer from '../components/CollapsibleAnswer.vue'
import AISummaryTab from '../components/AISummaryTab.vue'
import type { FollowUpEntry } from '../components/FollowUpBlock.vue'
import ImageGrid from '../components/ImageGrid.vue'
import MapCard from '../components/MapCard.vue'
import WebResultList from '../components/WebResultList.vue'
import { useSiteStore } from '../stores/site'
import { ask, askUrl, createNote, getSession, ApiError, type AskMode, type ImageItem, type MapPoint, type Source } from '../services/api'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'
import { clearSearchSession } from '../composables/useSearchSession'
import { settleAskGhost } from '../services/motion'
import { resolveLocationForQuery } from '../composables/useDeviceLocation'
import { needsLocationContext } from '../utils/locationIntent'
import { loadSearchSession, saveSearchSession, toFollowUpEntries } from '../composables/useSearchSession'
import { mapChatSession } from '../utils/mapChatSession'

const route = useRoute()
const router = useRouter()
const site = useSiteStore()
const auth = useAuthStore()
const toast = useToastStore()
const savingNote = ref(false)
onMounted(() => site.load())

const query = computed(() => String(route.query.q || '').trim())
const urlSessionId = computed(() => String(route.query.session || '').trim())
// Ask mode: 'enhanced' runs the AI pipeline (default), 'search' returns raw web results.
const mode = ref<AskMode>(route.query.mode === 'search' ? 'search' : 'enhanced')
// Result tabs: All shows the web results, AI Summary hosts the AI overview +
// follow-up conversation, Images the image grid.
const activeTab = ref<'all' | 'ai' | 'images'>('all')
const loading = ref(false)
const restoring = ref(false)
const error = ref<string | null>(null)
const guestLimitReached = ref(false)
const answer = ref('')
const sources = ref<Source[]>([])
const images = ref<ImageItem[]>([])
const mapCenter = ref<MapPoint | null>(null)
const mapMarkers = ref<MapPoint[]>([])
const sessionId = ref<string | null>(null)
const elapsed = ref(0)
const thread = ref<FollowUpEntry[]>([])
const primaryCollapsed = ref(false)
const activeFollowUpId = ref<number | null>(null)
const urlLoading = ref(false)
const urlError = ref<string | null>(null)
const locating = ref(false)
const usedLocation = ref(false)
const locationMissing = ref(false)
let followUpSeq = 0

function messageOf(cause: unknown): string {
  return cause instanceof Error ? cause.message : 'Something went wrong.'
}

// The backend rejects anonymous callers who already used their single AI
// search (AISY02004) — surface a sign-in CTA instead of a retry loop.
function isGuestLimit(cause: unknown): boolean {
  return cause instanceof ApiError && cause.code === 'AISY02004'
}

function collapsePriorContent() {
  if (answer.value) primaryCollapsed.value = true
  thread.value.forEach((entry) => {
    if (entry.answer && !entry.loading) entry.collapsed = true
  })
}

function syncSessionUrl() {
  if (!sessionId.value || !query.value) return
  if (urlSessionId.value === sessionId.value) return
  router.replace({ path: '/search', query: { q: query.value, session: sessionId.value } })
}

function persistState() {
  if (!sessionId.value || !query.value || !answer.value) return
  saveSearchSession({
    sessionId: sessionId.value,
    query: query.value,
    answer: answer.value,
    sources: sources.value,
    images: images.value,
    mapCenter: mapCenter.value,
    mapMarkers: mapMarkers.value,
    thread: thread.value
      .filter((entry) => entry.answer && !entry.loading)
      .map((entry) => ({
        question: entry.question,
        answer: entry.answer,
        sources: entry.sources,
        images: entry.images,
        mapCenter: entry.mapCenter ?? null,
        mapMarkers: entry.mapMarkers || [],
        collapsed: entry.collapsed,
      })),
    primaryCollapsed: primaryCollapsed.value,
    elapsed: elapsed.value,
    savedAt: Date.now(),
  })
  syncSessionUrl()
}

function applyPersistedState(saved: ReturnType<typeof loadSearchSession>) {
  if (!saved) return
  sessionId.value = saved.sessionId
  answer.value = saved.answer
  sources.value = saved.sources
  images.value = saved.images || []
  mapCenter.value = saved.mapCenter ?? null
  mapMarkers.value = saved.mapMarkers || []
  primaryCollapsed.value = saved.primaryCollapsed
  elapsed.value = saved.elapsed
  thread.value = toFollowUpEntries(saved.thread)
  followUpSeq = thread.value.length
  error.value = null
}

async function restoreFromApi(id: string): Promise<boolean> {
  restoring.value = true
  try {
    const session = await getSession(id)
    const mapped = mapChatSession(session)
    if (!mapped.answer) return false
    sessionId.value = session.sessionId
    answer.value = mapped.answer
    sources.value = mapped.sources
    images.value = mapped.images
    mapCenter.value = mapped.mapCenter
    mapMarkers.value = mapped.mapMarkers
    thread.value = mapped.thread
    followUpSeq = mapped.followUpSeq
    error.value = null
    persistState()
    return true
  } catch {
    return false
  } finally {
    restoring.value = false
  }
}

function resetState() {
  loading.value = false
  restoring.value = false
  error.value = null
  guestLimitReached.value = false
  answer.value = ''
  sources.value = []
  images.value = []
  mapCenter.value = null
  mapMarkers.value = []
  sessionId.value = null
  elapsed.value = 0
  thread.value = []
  primaryCollapsed.value = false
  activeFollowUpId.value = null
  urlError.value = null
  locating.value = false
  usedLocation.value = false
  locationMissing.value = false
  followUpSeq = 0
  mode.value = route.query.mode === 'search' ? 'search' : 'enhanced'
  activeTab.value = 'all'
}

async function bootstrap() {
  if (!query.value) return

  const cached = loadSearchSession(query.value, urlSessionId.value || undefined)
  if (cached) {
    applyPersistedState(cached)
    syncSessionUrl()
    return
  }

  if (urlSessionId.value) {
    const restored = await restoreFromApi(urlSessionId.value)
    if (restored) return
  }

  await runInitial()
}

async function runInitial() {
  if (!query.value) return
  loading.value = true
  error.value = null
  locationMissing.value = false
  usedLocation.value = false
  const startedAt = performance.now()
  try {
    if (needsLocationContext(query.value)) locating.value = true
    const location = await resolveLocationForQuery(query.value)
    locating.value = false
    usedLocation.value = Boolean(location)
    if (needsLocationContext(query.value) && !location) locationMissing.value = true
    const result = await ask(query.value, undefined, location, mode.value)
    answer.value = result.answer
    sources.value = result.sources || []
    images.value = result.images || []
    mapCenter.value = result.mapCenter || null
    mapMarkers.value = result.mapMarkers || []
    sessionId.value = result.sessionId
    elapsed.value = Math.round((performance.now() - startedAt) / 100) / 10
    // AI overview ships collapsed ("envelope") by default so the highlighted
    // web results are the first thing people see; click to expand the summary.
    primaryCollapsed.value = true
    persistState()
  } catch (cause) {
    guestLimitReached.value = isGuestLimit(cause)
    error.value = messageOf(cause)
  } finally {
    loading.value = false
    locating.value = false
  }
}

async function followUp(question: string) {
  if (!sessionId.value) return

  collapsePriorContent()

  const id = ++followUpSeq
  activeFollowUpId.value = id
  thread.value.push({
    id,
    question,
    answer: '',
    sources: [],
    images: [],
    mapCenter: null,
    mapMarkers: [],
    error: null,
    loading: true,
    collapsed: false,
    highlighted: false,
  })

  const index = thread.value.length - 1
  await nextTick()

  try {
    let location
    if (needsLocationContext(question)) locating.value = true
    location = await resolveLocationForQuery(question)
    locating.value = false
    if (needsLocationContext(question) && !location) locationMissing.value = true
    else if (location) usedLocation.value = true
    const result = await ask(question, sessionId.value, location, mode.value)
    thread.value[index].answer = result.answer
    thread.value[index].sources = result.sources || []
    thread.value[index].images = result.images || []
    thread.value[index].mapCenter = result.mapCenter || null
    thread.value[index].mapMarkers = result.mapMarkers || []
    thread.value[index].loading = false
    thread.value[index].highlighted = true
    persistState()

    await nextTick()
    window.setTimeout(() => {
      const entry = thread.value.find((item) => item.id === id)
      if (entry) entry.highlighted = false
    }, 2400)
  } catch (cause) {
    thread.value[index].error = messageOf(cause)
    thread.value[index].loading = false
    if (isGuestLimit(cause)) guestLimitReached.value = true
  } finally {
    locating.value = false
    if (activeFollowUpId.value === id) activeFollowUpId.value = null
  }
}

function onAsk(question: string) {
  if (sessionId.value && question.trim() === query.value) return
  // The header box always starts a fresh search; follow-ups live in the
  // "AI Summary" tab, so there is no follow-up branch here anymore.
  clearSearchSession()
  resetState()
  router.push({ path: '/search', query: { q: question, mode: mode.value } })
}

// Switches a search-only result to the enhanced pipeline and re-runs it.
function upgradeToEnhanced() {
  if (mode.value === 'enhanced') return
  resetState()
  mode.value = 'enhanced'
  router.replace({ path: '/search', query: { q: query.value, mode: 'enhanced' } })
  void runInitial()
}

async function submitUrl(url: string) {
  urlLoading.value = true
  urlError.value = null
  try {
    const result = await askUrl(url)
    answer.value = result.answer
    sources.value = result.sources || []
    images.value = result.images || []
    mapCenter.value = result.mapCenter || null
    mapMarkers.value = result.mapMarkers || []
    sessionId.value = result.sessionId
    error.value = null
    primaryCollapsed.value = true
    persistState()
  } catch (cause) {
    if (isGuestLimit(cause)) {
      guestLimitReached.value = true
      error.value = messageOf(cause)
    } else {
      urlError.value = messageOf(cause)
    }
  } finally {
    urlLoading.value = false
  }
}

function setFollowUpCollapsed(id: number, collapsed: boolean) {
  const entry = thread.value.find((item) => item.id === id)
  if (entry) {
    entry.collapsed = collapsed
    persistState()
  }
}

// Mini-apps integration: save the current answer/summary as a note, linked
// back to the search it came from.
async function saveToNotes() {
  if (savingNote.value || !answer.value || !auth.isAuthed) return
  savingNote.value = true
  try {
    await createNote({
      title: query.value,
      content: answer.value,
      sourceQuery: query.value,
      ...(sessionId.value ? { sessionId: sessionId.value } : {}),
    })
    toast.success('Saved to your notes.')
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    savingNote.value = false
  }
}

onMounted(bootstrap)
onMounted(settleAskGhost)
onBeforeUnmount(persistState)
watch(
  () => [route.query.q, route.query.session] as const,
  (next, prev) => {
    if (!prev) return
    if (next[0] === prev[0] && next[1] === prev[1]) return
    resetState()
    void bootstrap()
  },
)
</script>

<template>
  <main class="page-shell result-page">
    <AppHeader compact>
      <template #center>
        <AskBox
          variant="google"
          placeholder="Ask a question, explore an idea…"
          :has-session="false"
          @submit="onAsk"
        />
      </template>
    </AppHeader>

    <section class="result-content">
      <div class="result-kicker">
        <span class="result-kind">New search</span>
        <span>·</span>
        <span>Search results</span>
        <span>·</span>
        <span>{{ loading || restoring ? (locating ? 'Getting your location…' : restoring ? 'Restoring your search…' : 'Searching the web…') : `${sources.length} sources${elapsed ? ` · ${elapsed}s` : ''}` }}</span>
        <template v-if="usedLocation">
          <span>·</span>
          <span class="result-location-tag">Using your location</span>
        </template>
        <button
          v-if="auth.isAuthed && answer && !loading && !restoring"
          type="button"
          class="save-note-btn"
          :disabled="savingNote"
          @click="saveToNotes"
        >
          {{ savingNote ? 'Saving…' : 'Save summary to notes' }}
        </button>
      </div>
      <p v-if="locationMissing" class="location-note">Location wasn't shared, so nearby results may be less accurate. Allow location access in your browser for better local answers.</p>
      <h1 class="result-query">{{ query || 'Your question' }}</h1>

      <div v-if="guestLimitReached && error && !loading && !restoring" class="guest-limit-banner" role="alert">
        <span class="guest-limit-copy">{{ error }}</span>
        <button type="button" class="base-button button-primary" @click="router.push('/login')">Sign in to keep searching</button>
      </div>
      <ErrorBanner v-else-if="error && !loading && !restoring" :message="error" @retry="runInitial" />

      <template v-if="loading || restoring">
        <ResearchLoading
          :label="restoring ? 'Restoring your search' : locating ? 'Getting your location' : mode === 'search' ? 'Searching the web' : 'Researching your question'"
          :note="restoring ? 'Loading your previous conversation and its sources…' : locating ? 'Locating you for nearby, more relevant results…' : mode === 'search' ? 'Finding and ranking the best results across the web…' : 'Reading the best sources and building your answer…'"
        />
      </template>

      <template v-else>
        <div class="result-tabs" role="tablist" aria-label="Result type">
          <button type="button" role="tab" :aria-selected="activeTab === 'all'" :class="{ active: activeTab === 'all' }" @click="activeTab = 'all'">All</button>
          <button v-if="mode === 'enhanced' || Boolean(answer)" type="button" role="tab" :aria-selected="activeTab === 'ai'" :class="{ active: activeTab === 'ai' }" @click="activeTab = 'ai'">{{ mode === 'enhanced' ? 'AI Summary' : 'Summary' }}</button>
          <button v-if="images.length" type="button" role="tab" :aria-selected="activeTab === 'images'" :class="{ active: activeTab === 'images' }" @click="activeTab = 'images'">Images</button>
        </div>

        <template v-if="activeTab === 'all'">
          <template v-if="mode === 'enhanced'">
            <section v-if="sources.length" class="web-search-section" aria-label="Web search results">
              <div class="web-search-head">
                <h2 class="web-search-title">
                  <svg class="web-search-icon" viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
                    <circle cx="11" cy="11" r="6.5" fill="none" stroke="currentColor" stroke-width="1.9" />
                    <path d="M15.8 15.8L20 20M11 7.8a3.2 3.2 0 0 1 3.2 3.2" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" />
                  </svg>
                  Web search
                </h2>
                <span class="web-search-count">{{ sources.length }} result{{ sources.length === 1 ? '' : 's' }}</span>
              </div>
              <p class="web-search-note">The best sources we found across the web — cited, ranked, and ready to explore.</p>
              <WebResultList :sources="sources" :query="query" heading="Top matches" :show-count="false" class="web-search-list" />
            </section>
          </template>

          <section v-else class="search-results-only">
            <div class="search-results-head">
              <div>
                <div class="section-label">Web results</div>
                <h2 class="search-results-title">Top matches</h2>
              </div>
              <button type="button" class="base-button button-secondary search-upgrade" @click="upgradeToEnhanced">Ask AI to write it</button>
            </div>
            <p class="search-only-note">Summary pulled from the top search results — no AI used. Ask AI for a synthesized, cited answer.</p>
            <CollapsibleAnswer
              v-if="answer"
              label="Summary from top results"
              :answer="answer"
              :sources="sources"
              :query="query"
              :collapsed="primaryCollapsed"
              :map-center="mapCenter"
              :map-markers="mapMarkers"
              @update:collapsed="(value) => { primaryCollapsed = value; persistState() }"
            />
            <template v-else-if="sources.length">
              <MapCard v-if="mapCenter || mapMarkers.length" :center="mapCenter" :markers="mapMarkers" />
              <WebResultList :sources="sources" :query="query" heading="Web results" />
            </template>
            <div v-if="!answer && !sources.length" class="empty-sources">
              <div class="empty-source-copy">
                <h2>No web results found</h2>
                <p>Try rewording your question, or submit a URL for the AI to read that page directly.</p>
              </div>
              <div class="url-ask-wrap">
                <UrlAskBox @submit="submitUrl" />
                <p v-if="urlLoading" class="empty-note">Reading that page…</p>
                <p v-else-if="urlError" class="empty-note empty-note--error">{{ urlError }}</p>
              </div>
            </div>
          </section>

          <section v-if="mode === 'enhanced' && !sources.length" class="empty-sources" :class="{ 'empty-sources--compact': Boolean(answer) }">
            <div class="empty-source-copy">
              <h2 v-if="!answer">No web sources yet</h2>
              <p>{{ answer ? 'No web sources for this answer. Submit a URL to read a specific page.' : 'Ask a different question, or submit a URL and the AI will read that page directly.' }}</p>
            </div>
            <div class="url-ask-wrap">
              <UrlAskBox @submit="submitUrl" />
              <p v-if="urlLoading" class="empty-note">Reading that page…</p>
              <p v-else-if="urlError" class="empty-note empty-note--error">{{ urlError }}</p>
            </div>
          </section>
        </template>

        <section v-else-if="activeTab === 'ai'" class="ai-tab-content">
          <AISummaryTab
            :answer="answer"
            :sources="sources"
            :map-center="mapCenter"
            :map-markers="mapMarkers"
            :collapsed="primaryCollapsed"
            :thread="thread"
            :active-follow-up-id="activeFollowUpId"
            :can-follow-up="Boolean(sessionId)"
            :search-only="mode === 'search'"
            @update:collapsed="(value) => { primaryCollapsed = value; persistState() }"
            @update:collapsed-followup="setFollowUpCollapsed"
            @follow-up="followUp"
          />
        </section>

        <section v-else-if="activeTab === 'images' && images.length" class="image-tab-content">
          <ImageGrid :images="images" />
        </section>
      </template>
    </section>
  </main>
</template>

<style scoped>
.search-results-only { min-width: 0; margin-top: 38px; padding: 0 0 28px; border-bottom: 1px solid var(--color-border); }
.web-search-section {
  min-width: 0;
  margin-top: 8px;
}
.web-search-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}
.web-search-title {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  margin: 0;
  font-size: 1.3rem;
  font-weight: 720;
  letter-spacing: -.03em;
}
.web-search-icon { flex: 0 0 auto; color: var(--color-primary); }
.web-search-count {
  flex: 0 0 auto;
  padding: 4px 10px;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-surface-subtle);
  color: var(--color-muted);
  font-size: .72rem;
  font-weight: 680;
  white-space: nowrap;
}
.web-search-note {
  margin: 8px 0 0;
  color: var(--color-muted);
  font-size: .82rem;
}
.web-search-list :deep(.web-results) { margin-top: 18px; }
.web-search-list :deep(.sources-heading) { display: none; }
.search-results-head { display: flex; align-items: end; justify-content: space-between; gap: 18px; }
.search-results-title { margin: 6px 0 0; font-size: 1.35rem; letter-spacing: -.03em; }
.search-upgrade { flex: 0 0 auto; }
.search-only-note { margin: 10px 0 0; color: var(--color-muted); font-size: .82rem; }
.search-results-only :deep(.web-results) { margin-top: 22px; }
@media (max-width: 520px) { .search-results-head { align-items: flex-start; flex-direction: column; } }
.empty-sources--compact {
  margin-top: 8px;
  padding-top: 18px;
  border-top: 0;
}
.guest-limit-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 20px;
  padding: 14px 18px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 26%, var(--color-border));
  border-radius: 14px;
  background: color-mix(in srgb, var(--color-primary) 6%, var(--color-surface));
  color: var(--color-text);
  font-size: .9rem;
}
.guest-limit-copy { line-height: 1.5; }
@media (max-width: 520px) { .guest-limit-banner { align-items: flex-start; flex-direction: column; } }
.save-note-btn {
  margin-left: auto;
  padding: 6px 12px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 35%, var(--color-border));
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-primary) 8%, var(--color-surface));
  color: var(--color-primary);
  font-size: .72rem;
  font-weight: 700;
  cursor: pointer;
  transition: border-color .16s ease, background .16s ease;
}
.save-note-btn:hover:not(:disabled) { border-color: var(--color-primary); background: color-mix(in srgb, var(--color-primary) 14%, var(--color-surface)); }
.save-note-btn:disabled { opacity: .55; cursor: not-allowed; }
.result-kind {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 30%, var(--color-border));
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-primary) 7%, var(--color-surface));
  color: var(--color-primary);
  font-size: .66rem;
  font-weight: 760;
  letter-spacing: .06em;
  text-transform: uppercase;
}
.result-tabs {
  display: flex;
  gap: 2px;
  margin-top: 20px;
  border-bottom: 1px solid var(--color-border);
}
.result-tabs button {
  position: relative;
  padding: 10px 14px 12px;
  border: 0;
  background: transparent;
  color: var(--color-muted);
  font-size: .84rem;
  font-weight: 640;
  cursor: pointer;
  transition: color .16s ease;
}
.result-tabs button:hover { color: var(--color-text); }
.result-tabs button.active { color: var(--color-primary); }
.result-tabs button.active::after {
  content: '';
  position: absolute;
  right: 10px;
  bottom: -1px;
  left: 10px;
  height: 3px;
  border-radius: 3px 3px 0 0;
  background: var(--color-primary);
}
.image-tab-content { padding-top: 6px; }
.ai-tab-content { padding-top: 6px; }
.result-location-tag { color: var(--color-primary); font-weight: 680; }
.location-note {
  margin: 12px 0 0;
  padding: 10px 14px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 22%, var(--color-border));
  border-radius: 12px;
  background: color-mix(in srgb, var(--color-primary) 4%, var(--color-surface));
  color: var(--color-muted);
  font-size: .78rem;
  line-height: 1.5;
}
</style>
