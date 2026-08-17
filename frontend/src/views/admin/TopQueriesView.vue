<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getUserStats, type TopQuery } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import PageHeader from '../../components/PageHeader.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import EmptyState from '../../components/EmptyState.vue'

const toast = useToastStore()
const rows = ref<TopQuery[]>([]); const loading = ref(true)
const max = () => Math.max(1, ...rows.value.map((q) => q.count))
async function load() {
  loading.value = true
  try { rows.value = (await getUserStats()).topQueries } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
onMounted(load)
</script>
<template>
  <div>
    <PageHeader eyebrow="Statistics" title="Top queries" description="The most-asked questions across the platform." />
    <section class="admin-card">
      <div v-if="loading" class="admin-loading"><LoadingSpinner /></div>
      <ul v-else-if="rows.length" class="rank-list rank-list--full">
        <li v-for="(q, i) in rows" :key="q.query">
          <span class="rank-index">{{ i + 1 }}</span>
          <span class="rank-bar" :style="{ width: `${(q.count / max()) * 100}%` }"></span>
          <span class="rank-query">{{ q.query }}</span><span class="rank-count">{{ q.count }}</span>
        </li>
      </ul>
      <EmptyState v-else title="No queries yet" message="Questions will appear here as users ask the assistant." />
    </section>
  </div>
</template>