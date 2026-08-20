<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getPublicMiniApp, type MiniApp } from '../services/api'
import LoadingSpinner from '../components/LoadingSpinner.vue'
import MiniAppFrame from '../components/apps/MiniAppFrame.vue'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const loading = ref(true)
const error = ref('')
const app = ref<MiniApp | null>(null)

const slug = computed(() => route.params.slug as string)

const source = computed(() => {
  if (!app.value) return { html: '', css: '', js: '' }
  return { html: app.value.html, css: app.value.css, js: app.value.js }
})

function goBack() {
  if (auth.isAuthed) router.push('/apps')
  else router.push('/')
}

async function loadApp() {
  loading.value = true
  error.value = ''
  try {
    app.value = await getPublicMiniApp(slug.value)
  } catch (e: any) {
    error.value = e?.message || 'Could not load that mini app.'
  } finally {
    loading.value = false
  }
}

onMounted(loadApp)
</script>

<template>
  <main class="page-shell runner-page">
    <div class="runner-head">
      <button type="button" class="base-button button-secondary runner-back" @click="goBack">← Back</button>
      <template v-if="app">
        <span class="runner-title">{{ app.icon }} {{ app.name }}</span>
        <span v-if="app.description" class="runner-desc">{{ app.description }}</span>
      </template>
    </div>

    <div class="runner-body">
      <LoadingSpinner v-if="loading" />
      <div v-else-if="error" class="runner-error">
        <p>{{ error }}</p>
        <button type="button" class="base-button button-primary" @click="loadApp">Try again</button>
      </div>
      <MiniAppFrame v-else-if="app" :source="source" :title="app.name" />
    </div>
  </main>
</template>

<style scoped>
.runner-page { padding-top: 0; display: grid; grid-template-rows: auto 1fr; min-height: 100dvh; }

.runner-head {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 24px;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface);
}
.runner-back { flex: 0 0 auto; font-size: .82rem; }
.runner-title { font-weight: 720; font-size: .95rem; white-space: nowrap; }
.runner-desc {
  color: var(--color-muted);
  font-size: .8rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.runner-body {
  display: grid;
  place-items: center;
  min-height: 0;
}
.runner-body :deep(.mini-app-frame) {
  width: 100%;
  height: 100%;
  min-height: calc(100dvh - 60px);
}

.runner-error {
  display: grid;
  gap: 12px;
  text-align: center;
}
.runner-error p { margin: 0; color: var(--color-muted); font-size: .9rem; }
</style>
