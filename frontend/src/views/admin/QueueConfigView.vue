<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { getQueueConfig, updateQueueConfig, type QueueConfig } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import PageHeader from '../../components/PageHeader.vue'
import FormField from '../../components/FormField.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'

const toast = useToastStore()
const config = ref<QueueConfig | null>(null)
const loading = ref(true); const saving = ref(false)
const draft = ref({ maxConcurrent: 4, maxQueueSize: 100, requestTimeoutMs: 120000, perUserRateLimit: 10, suggestionCacheHours: 6, defaultDailyQuota: 3, maxImageResults: 20 })

async function load() {
  loading.value = true
  try { config.value = await getQueueConfig() } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
async function save() {
  saving.value = true
  try {
    config.value = await updateQueueConfig({ ...draft.value, maxConcurrent: Number(draft.value.maxConcurrent), maxQueueSize: Number(draft.value.maxQueueSize), requestTimeoutMs: Number(draft.value.requestTimeoutMs), perUserRateLimit: Number(draft.value.perUserRateLimit), suggestionCacheHours: Number(draft.value.suggestionCacheHours), defaultDailyQuota: Number(draft.value.defaultDailyQuota), maxImageResults: Number(draft.value.maxImageResults) })
    toast.success('Queue settings applied immediately.')
  } catch (e) { toast.error((e as Error).message) } finally { saving.value = false }
}
onMounted(async () => { await load(); if (config.value) Object.assign(draft.value, config.value) })

// The summary strip reflects the live draft so it updates as the owner types.
const summary = computed(() => [
  { label: 'Max concurrent', value: String(draft.value.maxConcurrent) },
  { label: 'Queue size', value: String(draft.value.maxQueueSize) },
  { label: 'Request timeout', value: formatTimeout(draft.value.requestTimeoutMs) },
  { label: 'Per-user rate', value: draft.value.perUserRateLimit ? `${draft.value.perUserRateLimit}/min` : 'Unlimited' },
])
function formatTimeout(ms: number): string {
  return ms >= 60000 ? `${Math.round(ms / 60000)} min` : `${Math.round(ms / 1000)} s`
}
</script>
<template>
  <div>
    <PageHeader eyebrow="AI providers" title="Queue & limits" description="Concurrency and rate-limit knobs for the AI worker pool. Changes apply without redeploying." />
    <section v-if="loading" class="admin-loading"><LoadingSpinner /></section>
    <template v-else>
      <section class="queue-summary">
        <div v-for="s in summary" :key="s.label" class="latency-cell"><span>{{ s.label }}</span><strong>{{ s.value }}</strong></div>
      </section>

      <section class="admin-card admin-card--form">
        <div class="setting-title"><div><div class="section-label">Worker pool</div><h2>Concurrency</h2><p>How many answers are generated in parallel and how long each may take.</p></div></div>
        <div class="form-grid-2">
          <FormField label="Max concurrent" hint="Answers processed at once across the pool">
            <input v-model.number="draft.maxConcurrent" type="number" class="text-input" min="1" required />
          </FormField>
          <FormField label="Max queue size" hint="Pending requests held before overflow is rejected">
            <input v-model.number="draft.maxQueueSize" type="number" class="text-input" min="0" required />
          </FormField>
          <FormField label="Request timeout (ms)" hint="Total answer generation budget">
            <input v-model.number="draft.requestTimeoutMs" type="number" class="text-input" min="1000" required />
          </FormField>
        </div>

        <div class="setting-title"><div><div class="section-label">Limits &amp; quotas</div><h2>Rate limits</h2><p>How much AI usage each user gets, and how many images a search may return.</p></div></div>
        <div class="form-grid-2">
          <FormField label="Per-user rate limit" hint="Requests per user per minute">
            <input v-model.number="draft.perUserRateLimit" type="number" class="text-input" min="1" required />
          </FormField>
          <FormField label="Default daily quota" hint="AI usage per day granted to newly registered accounts (0 = unlimited). Existing users keep their own quota.">
            <input v-model.number="draft.defaultDailyQuota" type="number" class="text-input" min="0" required />
          </FormField>
          <FormField label="Max image results" hint="Image results shown per search (0 = unlimited). Follow-up asks never fetch images.">
            <input v-model.number="draft.maxImageResults" type="number" class="text-input" min="0" required />
          </FormField>
        </div>

        <div class="setting-title"><div><div class="section-label">Suggestions</div><h2>Main-page suggestions</h2><p>How long AI-composed follow-up suggestions are reused per user before being recomposed.</p></div></div>
        <div class="form-grid-2">
          <FormField label="Suggestion cache (hours)" hint="0 = always compose fresh">
            <input v-model.number="draft.suggestionCacheHours" type="number" class="text-input" min="0" required />
          </FormField>
        </div>

        <p class="form-hint">When the queue is full, users see a friendly “try again in a moment” message rather than an error.</p>
        <div class="modal-submit-row">
          <button class="base-button button-primary" :disabled="saving" @click="save">{{ saving ? 'Saving…' : 'Apply settings' }}</button>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.queue-summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin-bottom: 22px; }
.setting-title { display: flex; align-items: start; justify-content: space-between; gap: 16px; padding-top: 2px; }
.setting-title h2 { margin: 4px 0 0; font-size: 1rem; letter-spacing: -.02em; }
.setting-title p { margin: 4px 0 0; color: var(--color-muted); font-size: .8rem; line-height: 1.5; }
@media (max-width: 640px) { .queue-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
