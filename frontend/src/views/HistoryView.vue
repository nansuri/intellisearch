<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppHeader from '../components/AppHeader.vue'
import ConfirmModal from '../components/ConfirmModal.vue'
import { useToastStore } from '../stores/toast'
import { getHistory, clearHistory, type HistoryItem } from '../services/api'

const router = useRouter()
const toast = useToastStore()
const items = ref<HistoryItem[]>([])
const loading = ref(true)
const error = ref(false)
const clearing = ref(false)
const confirmClear = ref(false)

function timeAgo(iso: string): string {
  const seconds = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 1000))
  if (seconds < 60) return 'just now'
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.round(hours / 24)
  if (days < 30) return `${days}d ago`
  const months = Math.round(days / 30)
  return `${months}mo ago`
}

async function load() {
  loading.value = true
  error.value = false
  try {
    items.value = (await getHistory(100)).items
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

async function clearAll() {
  clearing.value = true
  try {
    await clearHistory()
    items.value = []
    confirmClear.value = false
    toast.success('Search history cleared.')
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    clearing.value = false
  }
}

function runSearch(query: string) {
  router.push({ path: '/search', query: { q: query } })
}

onMounted(load)
</script>

<template>
  <main class="page-shell account-page">
    <AppHeader />
    <section class="account-content">
      <div class="account-heading">
        <div>
          <div class="eyebrow"><span></span> History</div>
          <h1>Search history</h1>
          <p>Every search you've run, with a summary of the answer.</p>
        </div>
        <button v-if="items.length" class="base-button button-secondary" @click="confirmClear = true">Clear history</button>
      </div>

      <section v-if="loading" class="settings-card"><div class="admin-loading"><span class="spinner"></span></div></section>

      <section v-else-if="error" class="settings-card">
        <div class="setting-title">
          <div><div class="section-label">Oops</div><h2>Couldn't load your history</h2><p>Something went wrong while fetching your past searches.</p></div>
        </div>
        <div class="modal-submit-row">
          <button class="base-button button-primary" @click="load">Try again</button>
        </div>
      </section>

      <section v-else-if="!items.length" class="settings-card">
        <div class="setting-title">
          <div><div class="section-label">Empty</div><h2>No searches yet</h2><p>When you search from the main page, your questions and answer summaries will appear here.</p></div>
        </div>
        <div class="modal-submit-row">
          <button class="base-button button-primary" @click="router.push('/')">Search now</button>
        </div>
      </section>

      <section v-else class="settings-card">
        <div class="setting-title">
          <div><div class="section-label">Past searches</div><h2>{{ items.length }} search{{ items.length === 1 ? '' : 'es' }}</h2><p>Click a search to run it again. Summaries are pulled from your chat sessions on demand.</p></div>
        </div>
        <ul class="history-list">
          <li v-for="item in items" :key="item.id" class="history-item">
            <button type="button" class="history-main" :title="`Run “${item.query}” again`" @click="runSearch(item.query)">
              <span class="history-query">{{ item.query }}</span>
              <span v-if="item.summary" class="history-summary">{{ item.summary }}</span>
              <span v-else class="history-summary history-summary--muted">Web results only — no AI summary.</span>
            </button>
            <span class="history-time">{{ timeAgo(item.createdAt) }}</span>
          </li>
        </ul>
      </section>
    </section>

    <ConfirmModal
      v-if="confirmClear"
      :open="true"
      title="Clear search history"
      message="This permanently removes all of your past searches and the suggestions based on them."
      confirm-label="Clear history"
      :busy="clearing"
      @close="confirmClear = false"
      @confirm="clearAll"
    />
  </main>
</template>

<style scoped>
.history-list { display: grid; margin: 22px 0 0; padding: 0; list-style: none; }
.history-item { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 14px 2px; border-bottom: 1px solid var(--color-border); }
.history-item:last-child { border-bottom: 0; }
.history-main { display: grid; gap: 5px; min-width: 0; padding: 0; border: 0; background: transparent; color: var(--color-text); cursor: pointer; text-align: left; font: inherit; }
.history-main:hover .history-query { color: var(--color-primary); }
.history-query { font-size: .95rem; font-weight: 700; letter-spacing: -.01em; transition: color .14s ease; }
.history-summary {
  display: -webkit-box;
  overflow: hidden;
  color: var(--color-muted);
  font-size: .82rem;
  line-height: 1.5;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
.history-summary--muted { font-style: italic; }
.history-time { flex-shrink: 0; align-self: flex-start; padding-top: 3px; color: var(--color-muted); font-size: .74rem; }
@media (max-width: 520px) { .history-item { align-items: flex-start; flex-direction: column; gap: 4px; } }
</style>
