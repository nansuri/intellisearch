<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getUserStats, type PerUserUsage } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import PageHeader from '../../components/PageHeader.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import EmptyState from '../../components/EmptyState.vue'

const toast = useToastStore()
const rows = ref<PerUserUsage[]>([]); const loading = ref(true)
const max = () => Math.max(1, ...rows.value.map((u) => u.count))
async function load() {
  loading.value = true
  try { rows.value = (await getUserStats()).perUserUsage } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
onMounted(load)
</script>
<template>
  <div>
    <PageHeader eyebrow="Statistics" title="Per-user usage" description="Questions asked per account. Sortable by activity." />
    <section class="admin-card">
      <div v-if="loading" class="admin-loading"><LoadingSpinner /></div>
      <div v-else-if="rows.length" class="user-table-wrap">
        <table class="user-table">
          <thead><tr><th>User</th><th>Email</th><th class="cell-num">Questions</th><th style="min-width:220px">Usage</th></tr></thead>
          <tbody>
            <tr v-for="u in rows" :key="u.userId">
              <td><strong>{{ u.name }}</strong></td>
              <td class="cell-muted">{{ u.email }}</td>
              <td class="cell-num">{{ u.count }}</td>
              <td><span class="usage-track"><span class="usage-fill" :style="{ width: `${(u.count / max()) * 100}%` }"></span></span></td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-else title="No activity yet" message="Per-user usage will appear here." />
    </section>
  </div>
</template>