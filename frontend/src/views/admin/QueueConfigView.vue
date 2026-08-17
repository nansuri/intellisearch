<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getQueueConfig, updateQueueConfig, type QueueConfig } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import PageHeader from '../../components/PageHeader.vue'
import FormField from '../../components/FormField.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'

const toast = useToastStore()
const config = ref<QueueConfig | null>(null)
const loading = ref(true); const saving = ref(false)
const draft = ref({ maxConcurrent: 4, maxQueueSize: 100, requestTimeoutMs: 120000, perUserRateLimit: 10, suggestionCacheHours: 6, defaultDailyQuota: 3 })

async function load() {
  loading.value = true
  try { config.value = await getQueueConfig() } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
async function save() {
  saving.value = true
  try {
    config.value = await updateQueueConfig({ ...draft.value, maxConcurrent: Number(draft.value.maxConcurrent), maxQueueSize: Number(draft.value.maxQueueSize), requestTimeoutMs: Number(draft.value.requestTimeoutMs), perUserRateLimit: Number(draft.value.perUserRateLimit), suggestionCacheHours: Number(draft.value.suggestionCacheHours), defaultDailyQuota: Number(draft.value.defaultDailyQuota) })
    toast.success('Queue settings applied immediately.')
  } catch (e) { toast.error((e as Error).message) } finally { saving.value = false }
}
onMounted(async () => { await load(); if (config.value) Object.assign(draft.value, config.value) })
</script>
<template>
  <div>
    <PageHeader eyebrow="AI providers" title="Queue & limits" description="Concurrency and rate-limit knobs for the AI worker pool. Changes apply without redeploying." />
    <section v-if="loading" class="admin-loading"><LoadingSpinner /></section>
    <section v-else class="admin-card admin-card--form">
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
        <FormField label="Per-user rate limit" hint="Requests per user per minute">
          <input v-model.number="draft.perUserRateLimit" type="number" class="text-input" min="1" required />
        </FormField>
        <FormField label="Suggestion cache (hours)" hint="How long AI-composed suggestions on the main page are reused before being recomposed (0 = always compose fresh)">
          <input v-model.number="draft.suggestionCacheHours" type="number" class="text-input" min="0" required />
        </FormField>
        <FormField label="Default daily quota" hint="AI usage per day granted to newly registered accounts (0 = unlimited). Existing users keep their own quota.">
          <input v-model.number="draft.defaultDailyQuota" type="number" class="text-input" min="0" required />
        </FormField>
      </div>
      <p class="form-hint">When the queue is full, users see a friendly “try again in a moment” message rather than an error.</p>
      <div class="modal-submit-row">
        <button class="base-button button-primary" :disabled="saving" @click="save">{{ saving ? 'Saving…' : 'Apply settings' }}</button>
      </div>
    </section>
  </div>
</template>