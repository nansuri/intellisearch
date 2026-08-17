<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AskBox from '../components/AskBox.vue'
import AppHeader from '../components/AppHeader.vue'
import ErrorBanner from '../components/ErrorBanner.vue'
import UrlAskBox from '../components/UrlAskBox.vue'
import CollapsibleAnswer from '../components/CollapsibleAnswer.vue'
import FollowUpBlock, { type FollowUpEntry } from '../components/FollowUpBlock.vue'
import { useSiteStore } from '../stores/site'
import SourceCard from '../components/SourceCard.vue'
import { ask, askUrl, getSession, type AskMode, type Source } from '../services/api'
import { settleAskGhost } from '../services/motion'
import { resolveLocationForQuery } from '../composables/useDeviceLocation'
import { needsLocationContext } from '../utils/locationIntent'
import { loadSearchSession, saveSearchSession, toFollowUpEntries } from '../composables/useSearchSession'
import { mapChatSession } from '../utils/mapChatSession'

const route = useRoute()
const router = useRouter()
const site = useSiteStore()
onMounted(() => site.load())

const query = computed(() => String(route.query.q || '').trim())
const urlSessionId = computed(() => String(route.query.session || '').trim())
// Ask mode: 'enhanced' runs the AI pipeline (default), 'search' returns raw web results.
const mode = ref<AskMode>(route.query.mode === 'search' ? 'search' : 'enhanced')
const loading = ref(false)
const restoring = ref(false)
const error = ref<string | null>(null)
const answer = ref('')
const sources = ref<Source[]>([])
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
    thread: thread.value
      .filter((entry) => entry.answer && !entry.loading)
      .map((entry) => ({
        question: entry.question,
        answer: entry.answer,
        sources: entry.sources,
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
  answer.value = ''
  sources.value = []
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
    sessionId.value = result.sessionId
    elapsed.value = Math.round((performance.now() - startedAt) / 100) / 10
    persistState()
  } catch (cause) {
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
  } finally {
    locating.value = false
    if (activeFollowUpId.value === id) activeFollowUpId.value = null
  }
}

function onAsk(question: string) {
  if (sessionId.value && question.trim() === query.value) return
  if (sessionId.value) {
    followUp(question)
    return
  }
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
    sessionId.value = result.sessionId
    error.value = null
    primaryCollapsed.value = false
    persistState()
  } catch (cause) {
    urlError.value = messageOf(cause)
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
          :placeholder="sessionId ? 'Ask a follow-up…' : 'Ask a question, explore an idea…'"
          @submit="onAsk"
        />
      </template>
    </AppHeader>

    <section class="result-content">
      <div class="result-kicker">
        <span>Search results</span>
        <span>·</span>
        <span>{{ loading || restoring ? (locating ? 'Getting your location…' : restoring ? 'Restoring your search…' : 'Searching the web…') : `${sources.length} sources${elapsed ? ` · ${elapsed}s` : ''}` }}</span>
        <template v-if="usedLocation">
          <span>·</span>
          <span class="result-location-tag">Using your location</span>
        </template>
      </div>
      <p v-if="locationMissing" class="location-note">Location wasn't shared, so nearby results may be less accurate. Allow location access in your browser for better local answers.</p>
      <h1 class="result-query">{{ query || 'Your question' }}</h1>

      <ErrorBanner v-if="error && !loading && !restoring" :message="error" @retry="runInitial" />

      <article v-if="loading || restoring" class="summary-card">
        <div class="section-label">{{ mode === 'search' ? 'Web results' : 'AI overview' }}</div>
        <div class="skeleton-line" />
        <div class="skeleton-line skeleton-line--short" />
        <div class="skeleton-line" />
        <div class="summary-note">{{ restoring ? 'Loading your previous answer…' : mode === 'search' ? 'Searching the web…' : 'Searching the web and reading the best sources…' }}</div>
      </article>

      <CollapsibleAnswer
        v-else-if="mode === 'enhanced'"
        label="AI overview"
        :answer="answer"
        :sources="sources"
        :collapsed="primaryCollapsed"
        @update:collapsed="(value) => { primaryCollapsed = value; persistState() }"
      />

      <section v-else class="search-results-only">
        <div class="search-results-head">
          <div>
            <div class="section-label">Web results</div>
            <h2 class="search-results-title">Top matches</h2>
          </div>
          <button type="button" class="base-button button-secondary search-upgrade" @click="upgradeToEnhanced">Ask AI for a summary</button>
        </div>
        <p class="search-only-note">Raw web results — Enhanced Ask adds an AI-synthesized answer with citations.</p>
        <div v-if="sources.length" class="sources">
          <div class="sources-heading"><h2>Results</h2><span>{{ sources.length }} found</span></div>
          <SourceCard v-for="source in sources" :key="source.position" :source="source" />
        </div>
        <div v-else class="empty-sources">
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

      <section v-if="!loading && !restoring && mode === 'enhanced' && !sources.length && !primaryCollapsed" class="empty-sources" :class="{ 'empty-sources--compact': Boolean(answer) }">
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

      <section v-if="thread.length" class="follow-up-thread">
        <div v-if="thread.some((e) => e.loading || e.highlighted)" class="follow-up-thread-hint">
          <span class="live-dot" />
          New follow-up in progress — previous answers are summarized above.
        </div>
        <FollowUpBlock
          v-for="(entry, index) in thread"
          :key="entry.id"
          :entry="entry"
          :index="index"
          :active="entry.id === activeFollowUpId"
          :search-only="mode === 'search'"
          @update:collapsed="setFollowUpCollapsed(entry.id, $event)"
        />
      </section>
    </section>
  </main>
</template>

<style scoped>
.search-results-only { margin-top: 38px; padding: 0 0 28px; border-bottom: 1px solid var(--color-border); }
.search-results-head { display: flex; align-items: end; justify-content: space-between; gap: 18px; }
.search-results-title { margin: 6px 0 0; font-size: 1.35rem; letter-spacing: -.03em; }
.search-upgrade { flex: 0 0 auto; }
.search-only-note { margin: 10px 0 0; color: var(--color-muted); font-size: .82rem; }
.search-results-only .sources { margin-top: 22px; }
@media (max-width: 520px) { .search-results-head { align-items: flex-start; flex-direction: column; } }
.empty-sources--compact {
  margin-top: 8px;
  padding-top: 18px;
  border-top: 0;
}
.follow-up-thread-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  padding: 10px 14px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 24%, var(--color-border));
  border-radius: 12px;
  background: color-mix(in srgb, var(--color-primary) 5%, var(--color-surface));
  color: var(--color-muted);
  font-size: .78rem;
}
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
