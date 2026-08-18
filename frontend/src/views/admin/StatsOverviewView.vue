<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { getAIStats, getTrends, getTrendingWords, getUserStats, type AIStats, type Trends, type TrendingWords, type UserStats } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import PageHeader from '../../components/PageHeader.vue'
import StatCard from '../../components/StatCard.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import EmptyState from '../../components/EmptyState.vue'
import TrendChart from '../../components/TrendChart.vue'
import DonutChart from '../../components/DonutChart.vue'
import AdminAlert from '../../components/admin/AdminAlert.vue'
import { formatPercent } from '../../utils/format'

const toast = useToastStore()
const stats = ref<UserStats | null>(null)
const trends = ref<Trends | null>(null)
const aiStats = ref<AIStats | null>(null)
const words = ref<TrendingWords | null>(null)
const loading = ref(true)
const maxUser = (s: UserStats | null) => Math.max(1, ...(s?.perUserUsage.map((u) => u.count) || [1]))
const maxWord = (w: TrendingWords | null) => Math.max(1, ...(w?.overall.map((t) => t.count) || [1]))

async function load() {
  loading.value = true
  try {
    const [s, t, a, w] = await Promise.all([getUserStats(), getTrends(), getAIStats(), getTrendingWords()])
    stats.value = s; trends.value = t; aiStats.value = a; words.value = w
  } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
onMounted(load)

const alerts = computed(() => {
  const items: Array<{ title: string; message: string; tone: 'warning' | 'danger'; to: string }> = []
  if (stats.value?.failed) {
    items.push({
      title: `${stats.value.failed} failed question${stats.value.failed === 1 ? '' : 's'}`,
      message: 'Some asks did not complete successfully — review the AI service page for error details.',
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
  return items
})
</script>
<template>
  <div>
    <PageHeader eyebrow="Statistics" title="Overview" description="A snapshot of activity across the platform." />
    <div v-if="loading" class="admin-loading"><LoadingSpinner /></div>
    <template v-else-if="stats">
      <section v-if="alerts.length" class="admin-alerts">
        <AdminAlert v-for="(alert, index) in alerts" :key="index" v-bind="alert" />
      </section>

      <section class="stat-grid">
        <StatCard label="Questions today" :value="stats.questionsToday" hint="Across all active users" />
        <StatCard label="Questions this week" :value="stats.questionsWeek" />
        <StatCard label="Active users (7d)" :value="stats.activeUsersWeek" />
        <StatCard label="Failed questions" :value="stats.failed" hint="Rejected or errored requests" />
      </section>

      <section v-if="aiStats" class="admin-card">
        <div class="mini-head"><h2>AI health</h2><router-link class="admin-site-link" to="/admin/stats/ai">View AI service</router-link></div>
        <div class="ai-health-body">
          <DonutChart :value="aiStats.successRate" :label="formatPercent(aiStats.successRate)" :tone="aiStats.successRate < 90 ? 'danger' : 'primary'" />
          <div class="ai-health-meta">
            <div><span>Completed</span><strong>{{ aiStats.totalCompleted }}</strong></div>
            <div><span>Failed</span><strong>{{ aiStats.totalFailed }}</strong></div>
            <div><span>Queue</span><strong>{{ aiStats.queue.queueDepth }} / {{ aiStats.queue.maxConcurrent }}</strong></div>
          </div>
        </div>
      </section>

      <section v-if="trends" class="trend-grid">
        <TrendChart title="Questions — last 7 days" :points="trends.daily" />
        <TrendChart title="Questions — last 8 weeks" :points="trends.weekly" empty-message="No weekly activity recorded yet." />
      </section>

      <section class="admin-card">
        <div class="mini-head"><h2>Unique users & visitors</h2><router-link class="admin-site-link" to="/admin/stats/visitors">View details</router-link></div>
        <div class="visitor-summary">
          <div class="visitor-metric"><span>Registered accounts</span><strong>{{ stats.registeredUsers }}</strong></div>
          <div class="visitor-metric"><span>Anonymous AI visitors</span><strong>{{ stats.anonymousVisitors }}</strong></div>
          <div class="visitor-metric"><span>Register-page visitors</span><strong>{{ stats.registerPageVisitors }}</strong></div>
        </div>
        <p class="privacy-note">Comparing register-page visitors with registered accounts shows registration interest vs. conversion. Each visitor is counted once — replays of the register page are deduped.</p>
      </section>
      <section class="admin-card">
        <div class="mini-head"><h2>Most searched words</h2><router-link class="admin-site-link" to="/admin/stats/words">View trends</router-link></div>
        <ul v-if="words?.overall.length" class="rank-list">
          <li v-for="t in words.overall.slice(0, 5)" :key="t.word">
            <span class="rank-bar" :style="{ width: `${(t.count / maxWord(words)) * 100}%` }"></span>
            <span class="rank-query">{{ t.word }}</span><span class="rank-count">{{ t.count }}</span>
          </li>
        </ul>
        <EmptyState v-else title="No words yet" message="Search terms will appear here as users ask." />
      </section>
      <section class="admin-card">
        <div class="mini-head"><h2>Heaviest users</h2><router-link class="admin-site-link" to="/admin/stats/usage">View all</router-link></div>
        <ul v-if="stats.perUserUsage.length" class="rank-list">
          <li v-for="u in stats.perUserUsage.slice(0, 5)" :key="u.userId">
            <span class="rank-bar" :style="{ width: `${(u.count / maxUser(stats)) * 100}%` }"></span>
            <span class="rank-query">{{ u.name }}<small>{{ u.email }}</small></span><span class="rank-count">{{ u.count }}</span>
          </li>
        </ul>
        <EmptyState v-else title="No activity" message="User usage will appear here." />
      </section>
    </template>
  </div>
</template>

<style scoped>
.admin-alerts { display: grid; gap: 10px; margin-bottom: 18px; }
.visitor-summary { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
.visitor-metric { display: grid; gap: 4px; padding: 14px 16px; border: 1px solid var(--color-border); border-radius: 12px; background: color-mix(in srgb, var(--color-bg) 45%, transparent); }
.visitor-metric span { color: var(--color-muted); font-size: .72rem; font-weight: 600; letter-spacing: .04em; text-transform: uppercase; }
.visitor-metric strong { font-size: 1.5rem; letter-spacing: -.03em; }
.privacy-note { margin: 14px 0 0; padding-top: 12px; border-top: 1px solid var(--color-border); color: var(--color-muted); font-size: .76rem; line-height: 1.5; }
@media (max-width: 640px) { .visitor-summary { grid-template-columns: 1fr; } }
</style>
