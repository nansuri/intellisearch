<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { getAIStats, type AIStats } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import PageHeader from '../../components/PageHeader.vue'
import StatCard from '../../components/StatCard.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import EmptyState from '../../components/EmptyState.vue'

const toast = useToastStore()
const s = ref<AIStats | null>(null); const loading = ref(true); const filter = ref('')
const errorCodes = ref<string[]>([])

async function load() {
  loading.value = true
  try {
    const result = await getAIStats(filter.value)
    s.value = result
    errorCodes.value = Array.from(new Set([...errorCodes.value, ...result.errors.map((e) => e.errorCode)])).sort()
  } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
onMounted(load)
import { formatMs, formatPercent } from '../../utils/format'
const filterLabel = computed(() => errorCodes.value.find((code) => code === filter.value) || '')
function changeFilter() { load() }
</script>
<template>
  <div>
    <PageHeader eyebrow="Statistics" title="AI service" description="Provider health, latency, and error breakdown for the assistant." />
    <div v-if="loading" class="admin-loading"><LoadingSpinner /></div>
    <template v-else-if="s">
      <section class="stat-grid">
        <StatCard label="Success rate" :value="formatPercent(s.successRate)" />
        <StatCard label="Completed" :value="s.totalCompleted" />
        <StatCard label="Failed" :value="s.totalFailed" />
        <StatCard label="Avg latency" :value="formatMs(s.latency.averageMs)" hint="p50 / p95 / p99 below" />
      </section>
      <section class="admin-card">
        <div class="mini-head"><h2>Latency percentiles</h2></div>
        <div class="latency-grid">
          <div v-for="(label, key) in { p50: 'p50', p95: 'p95', p99: 'p99' }" :key="key" class="latency-cell"><span>{{ label }}</span><strong>{{ formatMs(s.latency[key as 'p50']) }}</strong></div>
        </div>
      </section>
      <section class="admin-card">
        <div class="mini-head"><h2>Providers</h2></div>
        <div v-if="s.providers.length" class="user-table-wrap">
          <table class="user-table">
            <thead><tr><th>Provider</th><th>Model</th><th class="cell-num">Completed</th><th class="cell-num">Total</th><th class="cell-num">Success rate</th></tr></thead>
            <tbody>
              <tr v-for="p in s.providers" :key="p.name">
                <td><strong>{{ p.name }}</strong></td>
                <td class="cell-muted">{{ p.model }}</td>
                <td class="cell-num">{{ p.successes }}</td>
                <td class="cell-num">{{ p.total }}</td>
                <td class="cell-num">{{ p.total ? formatPercent(p.rate) : '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <EmptyState v-else title="No provider data" message="Provider success data appears once requests are processed." />
      </section>
      <section class="admin-card">
        <div class="mini-head"><h2>Failures by error code</h2>
          <div v-if="errorCodes.length" class="filter-row">
            <select v-model="filter" class="text-input filter-select" aria-label="Filter by error type" @change="changeFilter">
              <option value="">All error types</option>
              <option v-for="code in errorCodes" :key="code" :value="code">{{ code }}</option>
            </select>
          </div>
        </div>
        <div v-if="s.errors.length" class="user-table-wrap">
          <table class="user-table">
            <thead><tr><th>Error</th><th class="cell-num">Count</th><th>Last seen</th></tr></thead>
            <tbody>
              <tr v-for="e in s.errors" :key="e.errorCode">
                <td><code>{{ e.errorCode }}</code> <small class="cell-muted">{{ e.errorMessage }}</small></td>
                <td class="cell-num">{{ e.count }}</td>
                <td class="cell-muted">{{ new Date(e.lastSeen).toLocaleString() }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <EmptyState v-else title="No failures" :message="filter ? `No '${filterLabel}' failures recorded.` : 'No error codes recorded for completed requests.'" />
      </section>
      <section class="admin-card">
        <div class="mini-head"><h2>Queue health</h2></div>
        <div class="latency-grid latency-grid--4">
          <div class="latency-cell"><span>In flight</span><strong>{{ s.queue.inFlight }} / {{ s.queue.maxConcurrent }}</strong></div>
          <div class="latency-cell"><span>Queued</span><strong>{{ s.queue.queueDepth }}</strong></div>
          <div class="latency-cell"><span>Rejected</span><strong>{{ s.queue.rejected }}</strong></div>
        </div>
      </section>
    </template>
  </div>
</template>