<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  downloadApiDocMarkdown,
  generateMiniApp,
  getApiDocSections,
  listMyMiniApps,
  type ApiDocSection,
  type MiniApp,
} from '../services/api'
import { useToastStore } from '../stores/toast'
import { useAuthStore } from '../stores/auth'
import PageHeader from '../components/PageHeader.vue'
import LoadingSpinner from '../components/LoadingSpinner.vue'
import EmptyState from '../components/EmptyState.vue'
import MarkdownView from '../components/MarkdownView.vue'

const router = useRouter()
const toast = useToastStore()
const auth = useAuthStore()

const apps = ref<MiniApp[]>([])
const loading = ref(true)
const docSections = ref<ApiDocSection[]>([])
const docsLoading = ref(true)

const prompt = ref('')
const generating = ref(false)

function openApp(id: string) { router.push({ path: `/apps/${id}` }) }

function loadApps() {
  loading.value = true
  listMyMiniApps()
    .then(({ items }) => { apps.value = items })
    .catch((e) => toast.error((e as Error).message))
    .finally(() => { loading.value = false })
}

function openDocs() {
  docsLoading.value = true
  getApiDocSections()
    .then(({ sections }) => { docSections.value = sections })
    .catch(() => { docSections.value = [] })
    .finally(() => { docsLoading.value = false })
}

async function generate() {
  if (!prompt.value.trim() || generating.value) return
  generating.value = true
  try {
    const app = await generateMiniApp(prompt.value.trim())
    toast.success('Your app was generated — refine it below.')
    router.push({ path: `/apps/${app.id}` })
  } catch (e) {
    toast.error((e as Error).message || 'Could not start generation.')
  } finally {
    generating.value = false
  }
}

async function downloadDocs() {
  try {
    const { text, filename } = await downloadApiDocMarkdown()
    const blob = new Blob([text], { type: 'text/markdown;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
    toast.success('API reference downloaded.')
  } catch (e) {
    toast.error((e as Error).message || 'Download failed.')
  }
}

onMounted(() => {
  loadApps()
  openDocs()
})

const generatedHint = computed(() => (auth.user ? `Hi ${auth.user.name.split(' ')[0]} — describe the app you want and the AI builds the first draft for you to refine.` : ''))
</script>

<template>
  <main class="page-shell studio-page">
    <div class="studio-head">
      <PageHeader eyebrow="Apps" title="Mini app studio" description="Build small apps from plain HTML, CSS and JS and run them sandboxed on this site — with the AI API available inside every app.">
        <div class="studio-head-actions">
          <button type="button" class="base-button button-secondary" @click="downloadDocs">Download API reference (.md)</button>
          <button type="button" class="base-button button-primary" @click="openApp('new')">+ New app</button>
        </div>
      </PageHeader>
    </div>

    <section class="studio-generate">
      <div class="studio-generate-copy">
        <h2>Build with AI</h2>
        <p>Describe a mini app and the AI writes the first draft — HTML, CSS and JS — which you can refine in the editor and publish for any visitor.</p>
        <p v-if="generatedHint" class="studio-generate-hint">{{ generatedHint }}</p>
      </div>
      <form class="studio-generate-form" @submit.prevent="generate">
        <input v-model="prompt" class="text-input" type="text" maxlength="300" placeholder="e.g. a daily-word-count timer" aria-label="Describe the mini app to build" />
        <button type="submit" class="base-button button-primary" :disabled="generating || !prompt.trim()">
          {{ generating ? 'Building…' : 'Generate' }}
        </button>
      </form>
    </section>

    <section class="studio-section">
      <div class="studio-section-title">Your apps</div>
      <div v-if="loading" class="studio-loading"><LoadingSpinner /></div>
      <div v-else-if="apps.length" class="studio-grid">
        <article v-for="app in apps" :key="app.id" class="studio-card" @click="openApp(app.id)">
          <div class="studio-card-icon" aria-hidden="true">{{ app.icon || '🧩' }}</div>
          <div class="studio-card-body">
            <div class="studio-card-title">{{ app.name }}</div>
            <p class="studio-card-desc">{{ app.description || 'No description yet.' }}</p>
          </div>
          <div class="studio-card-sup">
            <span class="studio-badge" :class="`studio-badge--${app.visibility}`">{{ app.visibility }}</span>
          </div>
        </article>
      </div>
      <EmptyState v-else icon="🧩" title="No apps yet" message="Create a blank app, or describe one and let the AI write the first draft.">
        <button type="button" class="base-button button-primary" @click="openApp('new')">Create your first app</button>
        <button type="button" class="base-button button-secondary" @click="prompt = 'a daily habit tracker'">
          Try “a habit tracker”
        </button>
      </EmptyState>
    </section>

    <section class="studio-section">
      <div class="studio-section-title">
        AI API reference
        <button type="button" class="studio-refresh-docs" title="Reload documentation" @click="openDocs">↻</button>
      </div>
      <div v-if="docsLoading" class="studio-loading"><LoadingSpinner /></div>
      <div v-else-if="docSections.length" class="studio-doc-grid">
        <details v-for="section in docSections" :key="section.section" class="studio-doc" :open="section.section === 'Overview'">
          <summary class="studio-doc-head">
            <h3>{{ section.section }}</h3>
            <span>{{ section.entries.length }} {{ section.entries.length === 1 ? 'entry' : 'entries' }}</span>
          </summary>
          <div v-for="entry in section.entries" :key="entry.title" class="studio-doc-entry">
            <div class="studio-doc-entry-title">
              <span v-if="entry.method" class="studio-badge studio-badge--method">{{ entry.method }}</span>
              {{ entry.title }}
            </div>
            <MarkdownView :content="entry.markdown" />
          </div>
        </details>
      </div>
      <p v-else class="studio-no-docs">No API reference available yet.</p>
    </section>
  </main>
</template>

<style scoped>
.studio-page { padding-top: 0; }
.studio-head { margin-top: 28px; }
.studio-head :deep(.admin-page-head) { margin-bottom: 22px; }
.studio-head-actions { display: flex; align-items: center; gap: 12px; }

.studio-generate {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(260px, 420px);
  gap: 20px;
  align-items: center;
  margin-bottom: 28px;
  padding: 22px;
  border: 1px solid var(--color-border);
  border-radius: 16px;
  background: linear-gradient(135deg, var(--color-surface), color-mix(in srgb, var(--color-primary) 8%, var(--color-surface)));
}
.studio-generate-copy h2 { margin: 0 0 6px; font-size: 1.05rem; letter-spacing: -.01em; }
.studio-generate-copy p { margin: 0; color: var(--color-muted); font-size: .86rem; line-height: 1.55; }
.studio-generate-hint { margin-top: 6px !important; font-size: .8rem; }
.studio-generate-form { display: flex; gap: 10px; }

.studio-section { margin-top: 30px; }
.studio-section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
  color: var(--color-muted);
  font-size: .7rem;
  font-weight: 720;
  letter-spacing: .08em;
  text-transform: uppercase;
}
.studio-refresh-docs {
  padding: 0 5px;
  border: 0;
  background: transparent;
  color: var(--color-muted);
  cursor: pointer;
  font-size: .9rem;
}
.studio-refresh-docs:hover { color: var(--color-primary); }
.studio-loading { display: grid; place-items: center; padding: 48px 0; }
.studio-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 14px; }
.studio-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  border: 1px solid var(--color-border);
  border-radius: 16px;
  background: var(--color-surface);
  box-shadow: 0 10px 30px var(--color-shadow);
  cursor: pointer;
  transition: border-color .16s ease, transform .16s ease;
}
.studio-card:hover { border-color: var(--color-primary); transform: translateY(-2px); }
.studio-card-icon {
  display: inline-grid;
  place-items: center;
  width: 44px;
  height: 44px;
  flex: 0 0 auto;
  border-radius: 12px;
  background: var(--color-surface-subtle);
  font-size: 1.3rem;
}
.studio-card-body { min-width: 0; flex: 1 1 auto; }
.studio-card-title { font-weight: 720; letter-spacing: -.01em; }
.studio-card-desc { margin: 3px 0 0; color: var(--color-muted); font-size: .8rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.studio-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: .66rem;
  font-weight: 700;
  letter-spacing: .04em;
  text-transform: uppercase;
}
.studio-badge--public { background: color-mix(in srgb, #2e9d5b 16%, transparent); color: #1f7a46; }
.studio-badge--private { background: color-mix(in srgb, var(--color-muted) 16%, transparent); color: var(--color-muted); }
.studio-badge--method { background: color-mix(in srgb, var(--color-primary) 16%, transparent); color: var(--color-primary); }
.studio-card-sup { flex: 0 0 auto; }

.studio-doc-grid { display: grid; gap: 12px; }
.studio-doc {
  border: 1px solid var(--color-border);
  border-radius: 14px;
  background: var(--color-surface);
  overflow: hidden;
}
.studio-doc-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 18px;
  cursor: pointer;
  list-style: none;
}
.studio-doc-head::-webkit-details-marker { display: none; }
.studio-doc-head h3 { margin: 0; font-size: .92rem; }
.studio-doc-head span { color: var(--color-muted); font-size: .74rem; }
.studio-doc-entry { padding: 14px 18px 18px; border-top: 1px solid var(--color-border); }
.studio-doc-entry-title { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; font-weight: 720; font-size: .88rem; }
.studio-doc-entry :deep(.markdown) { color: var(--color-muted); font-size: .84rem; line-height: 1.6; overflow-wrap: anywhere; }
.studio-no-docs { color: var(--color-muted); font-size: .88rem; }

@media (max-width: 760px) {
  .studio-generate { grid-template-columns: 1fr; }
  .studio-generate-form { flex-direction: column; }
}
</style>