<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getVisitorStats, type VisitorStats } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import PageHeader from '../../components/PageHeader.vue'
import StatCard from '../../components/StatCard.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import TrendChart from '../../components/TrendChart.vue'

const toast = useToastStore()
const stats = ref<VisitorStats | null>(null)
const loading = ref(true)

async function load() {
  loading.value = true
  try {
    stats.value = await getVisitorStats()
  } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
onMounted(load)
</script>
<template>
  <div>
    <PageHeader eyebrow="Statistics" title="Unique users & visitors" description="How many accounts exist, who actually uses the AI service, and how much interest the register page drives." />
    <div v-if="loading" class="admin-loading"><LoadingSpinner /></div>
    <template v-else-if="stats">
      <section class="stat-grid">
        <StatCard label="Registered accounts" :value="stats.registeredUsers.total" hint="All-time totals" />
        <StatCard label="Active users" :value="stats.activeUsers" hint="Accounts that have run a question" />
        <StatCard label="Active users (7d)" :value="stats.activeUsers7d" />
        <StatCard label="Anonymous AI visitors" :value="stats.anonymousVisitors.total" hint="Guests that used the AI once" />
        <StatCard label="Register-page visitors" :value="stats.registerPageVisits.total" hint="Unique visitors, replayed opens deduped" />
      </section>

      <section class="trend-grid">
        <TrendChart title="New registrations — last 7 days" :points="stats.registeredUsers.daily" />
        <TrendChart title="New registrations — last 8 weeks" :points="stats.registeredUsers.weekly" empty-message="No registrations in this window yet." />
        <TrendChart title="Anonymous AI visitors — last 7 days" :points="stats.anonymousVisitors.daily" />
        <TrendChart title="Anonymous AI visitors — last 8 weeks" :points="stats.anonymousVisitors.weekly" empty-message="No anonymous AI activity yet." />
        <TrendChart title="Register-page visitors — last 7 days" :points="stats.registerPageVisits.daily" />
        <TrendChart title="Register-page visitors — last 8 weeks" :points="stats.registerPageVisits.weekly" empty-message="Nobody has opened the register page yet." />
      </section>
    </template>
  </div>
</template>