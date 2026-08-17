<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AskBox from '../components/AskBox.vue'
import AppHeader from '../components/AppHeader.vue'
import { useSiteStore } from '../stores/site'
import { useAuthStore } from '../stores/auth'
import { getHistory, getHistorySuggestions } from '../services/api'
import { createAskGhost } from '../services/motion'
import { resolveLocationForQuery } from '../composables/useDeviceLocation'
import { needsLocationContext } from '../utils/locationIntent'
import type { AskMode } from '../services/api'

const router = useRouter()
const site = useSiteStore(); onMounted(() => site.load())
const auth = useAuthStore()
const askBox = ref<InstanceType<typeof AskBox> | null>(null)
const locating = ref(false)
const mode = ref<AskMode>('enhanced')

// Recent searches + AI suggestions (signed-in users only).
const recent = ref<string[]>([])
const suggestions = ref<string[]>([])
const historyLoading = ref(false)
const suggestionsLoading = ref(false)
const historyError = ref(false)
const signedIn = computed(() => Boolean(auth.token && auth.isAuthed))

// Suggestions are composed by the LLM, so the backend caches them per user for
// a configurable window (ai_queue_config.suggestion_cache_hours, editable from
// the Owner Control Panel → Queue & limits). The ↻ button forces a refresh.

function loadRecent() {
  if (!signedIn.value) return
  historyLoading.value = true
  historyError.value = false
  getHistory(30)
    .then(({ items }) => {
      const seen = new Set<string>()
      recent.value = items
        .map((item) => item.query.trim())
        .filter((query) => {
          if (!query || seen.has(query)) return false
          seen.add(query)
          return true
        })
        .slice(0, 6)
    })
    .catch(() => { historyError.value = true })
    .finally(() => { historyLoading.value = false })
}

async function loadSuggestions(force = false) {
  if (!signedIn.value) return
  suggestionsLoading.value = true
  try {
    const { suggestions: items } = await getHistorySuggestions(force)
    suggestions.value = items
  } catch {
    suggestions.value = []
  } finally {
    suggestionsLoading.value = false
  }
}

async function onAsk(question: string, askMode: AskMode = 'enhanced') {
  const el = askBox.value?.$el
  if (el instanceof HTMLElement) {
    createAskGhost(el.getBoundingClientRect(), question)
  }
  if (needsLocationContext(question)) locating.value = true
  try {
    await resolveLocationForQuery(question)
  } finally {
    locating.value = false
  }
  router.push({ path: '/search', query: { q: question, mode: askMode } })
}

onMounted(() => {
  site.load()
  if (signedIn.value) {
    loadRecent()
    loadSuggestions()
  }
})
</script>

<template>
  <main class="page-shell main-page">
    <AppHeader />
    <section class="hero">
      <div class="hero-brand"><h1>{{ site.settings?.siteName || 'Intellisearch' }}</h1></div>
      <AskBox ref="askBox" :show-prompt="false" mode-toggle :mode="mode" :helper-text="locating ? 'Getting your location for nearby results…' : ''" @update:mode="mode = $event" @submit="onAsk" />
      <section v-if="signedIn && (historyLoading || recent.length || suggestions.length || suggestionsLoading)" class="history-panel" aria-label="Search history and suggestions">
        <div v-if="recent.length" class="history-group">
          <span class="history-label">Recent searches</span>
          <div class="prompt-list">
            <button v-for="query in recent" :key="query" type="button" @click="onAsk(query)">{{ query }}</button>
          </div>
        </div>
        <div v-if="suggestionsLoading" class="history-group">
          <span class="history-label">Composing suggestions…</span>
        </div>
        <div v-else-if="suggestions.length" class="history-group">
          <span class="history-label">
            Suggested for you
            <button class="history-refresh" type="button" title="Compose new suggestions" @click="loadSuggestions(true)">↻</button>
          </span>
          <div class="prompt-list">
            <button v-for="query in suggestions" :key="query" type="button" @click="onAsk(query)">{{ query }}</button>
          </div>
        </div>
        <p v-if="historyError" class="history-note">Couldn't load recent searches.</p>
      </section>
    </section>
  </main>
</template>

<style scoped>
.history-panel { display: grid; gap: 16px; max-width: 720px; margin: 22px auto 0; text-align: left; }
.history-group { display: grid; gap: 8px; }
.history-label { color: var(--color-muted); font-size: .72rem; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; }
.history-refresh { margin-left: 6px; padding: 0 4px; border: 0; background: transparent; color: var(--color-muted); cursor: pointer; font-size: .85rem; }
.history-refresh:hover { color: var(--color-primary); }
.history-note { margin: 0; color: var(--color-muted); font-size: .78rem; }
@media (max-width: 520px) { .history-panel { margin-top: 18px; } }
</style>
