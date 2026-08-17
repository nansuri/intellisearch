<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { getAIStats, getQueueConfig, getTrends, getUserStats, listProviders, type AIStats, type QueueConfig, type Trends, type UserStats } from '../../services/api'
import { useSiteStore } from '../../stores/site'
import { useToastStore } from '../../stores/toast'
import PageHeader from '../../components/PageHeader.vue'
import StatCard from '../../components/StatCard.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import EmptyState from '../../components/EmptyState.vue'
import TrendChart from '../../components/TrendChart.vue'
import AdminAlert from '../../components/admin/AdminAlert.vue'
import AdminIcon from '../../components/admin/AdminIcon.vue'
import { formatMs, formatPercent } from '../../utils/format'

const site = useSiteStore(); site.load()
const toast = useToastStore()
const stats = ref<UserStats | null>(null)
const aiStats = ref<AIStats | null>(null)
const trends = ref<Trends | null>(null)
const queue = ref<QueueConfig | null>(null)
const providers = ref<{ id: string; name: string; model: string; isActive: boolean }[]>([])
const loading = ref(true)

async function load() {
  loading.value = true
  try {
    const [s, a, t, q, p] = await Promise.all([getUserStats(), getAIStats(), getTrends(), getQueueConfig(), listProviders()])
    stats.value = s; aiStats.value = a; trends.value = t; queue.value = q; providers.value = p.providers
  } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
onMounted(load)

const tiles = [
  { to: '/admin/users', label: 'User management', desc: 'Search, create, edit, suspend, and delete accounts.', icon: 'user-list' },
  { to: '/admin/users/suspended', label: 'Suspended users', desc: 'Review and reinstate suspended accounts.', icon: 'suspended' },
  { to: '/admin/stats', label: 'Statistics', desc: 'Platform activity, trends, and top queries.', icon: 'chart' },
  { to: '/admin/stats/usage', label: 'Per-user usage', desc: 'Who asks the most, and who fails the most.', icon: 'usage' },
  { to: '/admin/stats/ai', label: 'AI service', desc: 'Success rate, errors, latency, and queue health.', icon: 'robot' },
  { to: '/admin/ai/providers', label: 'AI providers', desc: 'Configure Ollama or OpenAI-compatible providers.', icon: 'settings' },
  { to: '/admin/ai/queue', label: 'Queue & limits', desc: 'Concurrency, queue size, timeout, and rate limits.', icon: 'queue' },
  { to: '/admin/branding/identity', label: 'Site identity', desc: 'Dashboard name and tagline for the public page.', icon: 'tag' },
  { to: '/admin/branding/logo', label: 'Logo', desc: 'Upload, replace, or remove the public logo.', icon: 'image' },
]

const alerts = computed(() => {
  const items: Array<{ title: string; message: string; tone: 'warning' | 'danger' | 'info'; to?: string }> = []
  if (stats.value?.failed) {
    items.push({
      title: `${stats.value.failed} failed question${stats.value.failed === 1 ? '' : 's'}`,
      message: 'Some asks did not complete successfully. Review AI service errors for details.',
      tone: 'warning',
      to: '/admin/stats/ai',
    })
  }
  if (aiStats.value?.totalFailed) {
    items.push({
      title: `${aiStats.value.totalFailed} AI failure${aiStats.value.totalFailed === 1 ? '' : 's'}`,
      message: `Success rate is ${formatPercent(aiStats.value.successRate)}. Check error codes and provider health.`,
      tone: aiStats.value.successRate < 90 ? 'danger' : 'warning',
      to: '/admin/stats/ai',
    })
  }
  if (aiStats.value?.errors.length) {
    const top = aiStats.value.errors[0]
    items.push({
      title: `Top error: ${top.errorCode}`,
      message: `${top.errorMessage} (${top.count} occurrence${top.count === 1 ? '' : 's'})`,
      tone: 'danger',
      to: '/admin/stats/ai',
    })
  }
  if (aiStats.value?.queue.rejected) {
    items.push({
      title: `${aiStats.value.queue.rejected} queue rejection${aiStats.value.queue.rejected === 1 ? '' : 's'}`,
      message: 'Requests were turned away because the queue was full. Consider raising concurrency or queue size.',
      tone: 'warning',
      to: '/admin/ai/queue',
    })
  }
  if (!providers.value.length) {
    items.push({
      title: 'No AI provider configured',
      message: 'Add a provider before users can receive answers.',
      tone: 'info',
      to: '/admin/ai/providers',
    })
  }
  return items
})
</script>
<template>
  <div>
    <PageHeader eyebrow="Control Panel" :title="(site.settings?.siteName || 'Control Panel')" description="Everything in one place — manage users, monitor AI usage, tune providers, and configure branding." />

    <div v-if="loading" class="admin-loading"><LoadingSpinner /></div>
    <template v-else>
      <section v-if="alerts.length" class="admin-alerts">
        <AdminAlert v-for="(alert, index) in alerts" :key="index" v-bind="alert" />
      </section>

      <section class="stat-grid">
        <StatCard label="Questions today" :value="stats?.questionsToday ?? '—'" hint="Across all active users" />
        <StatCard label="Active users (7d)" :value="stats?.activeUsersWeek ?? '—'" />
        <StatCard label="AI success rate" :value="aiStats ? formatPercent(aiStats.successRate) : '—'" hint="Completed vs. failed requests" />
        <StatCard label="Queue health" :value="queue ? `${queue.maxConcurrent} workers` : '—'" hint="Editable under Queue & limits" />
      </section>

      <section v-if="trends" class="trend-grid">
        <TrendChart title="Questions — last 7 days" :points="trends.daily" />
        <TrendChart title="Questions — last 8 weeks" :points="trends.weekly" empty-message="No weekly activity recorded yet." />
      </section>

      <section class="admin-card">
        <div class="mini-head">
          <h2>Modules</h2>
          <span class="admin-site-link">Pick a module to manage it</span>
        </div>
        <div class="dash-grid">
          <router-link v-for="t in tiles" :key="t.to" class="dash-tile" :to="t.to">
            <span class="dash-icon" aria-hidden="true"><AdminIcon :name="t.icon" :size="20" /></span>
            <strong>{{ t.label }}</strong>
            <span>{{ t.desc }}</span>
          </router-link>
        </div>
      </section>

      <section v-if="aiStats" class="admin-card">
        <div class="mini-head"><h2>AI service snapshot</h2><router-link class="admin-site-link" to="/admin/stats/ai">View all</router-link></div>
        <div class="latency-grid latency-grid--4">
          <div class="latency-cell"><span>Queue depth</span><strong>{{ aiStats.queue.queueDepth }} / {{ aiStats.queue.maxConcurrent }}</strong></div>
          <div class="latency-cell"><span>In flight</span><strong>{{ aiStats.queue.inFlight }}</strong></div>
          <div class="latency-cell"><span>Rejected</span><strong>{{ aiStats.queue.rejected }}</strong></div>
          <div class="latency-cell"><span>Avg response</span><strong>{{ formatMs(aiStats.latency.averageMs) }}</strong></div>
        </div>
        <p v-if="providers.length" class="dash-active-provider">
          <span class="status-badge status-badge--accent">Active provider</span>
          <span>{{ providers.find((p) => p.isActive)?.name || 'None selected' }} · {{ providers.find((p) => p.isActive)?.model || '' }}</span>
        </p>
        <EmptyState v-show="!providers.length" title="No providers configured" message="Add an AI provider under AI providers to start answering questions." />
      </section>
    </template>
  </div>
</template>

<style scoped>
.admin-alerts { display: grid; gap: 10px; margin-bottom: 18px; }
</style>
