<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getTrendingWords, type TrendingWords } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import PageHeader from '../../components/PageHeader.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import EmptyState from '../../components/EmptyState.vue'

const toast = useToastStore()
const window = ref<'daily' | 'weekly'>('daily')
const data = ref<TrendingWords | null>(null)
const loading = ref(true)

async function load() {
  loading.value = true
  try { data.value = await getTrendingWords(window.value) } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
onMounted(load)

function switchWindow(next: 'daily' | 'weekly') {
  if (next === window.value) return
  window.value = next
  void load()
}

// Chart: top 5 overall terms, one bar per time bucket.
const CHART_COLORS = ['#4f6ef7', '#8b6dff', '#0ea5e9', '#f59e0b', '#34d399']
const topWords = () => data.value?.overall.slice(0, 5) || []
const chartMax = () => Math.max(1, ...data.value?.buckets.flatMap((b) => b.top.map((t) => t.count)) || [1])
const countFor = (bucketLabel: string, word: string) => data.value?.buckets.find((b) => b.label === bucketLabel)?.top.find((t) => t.word === word)?.count || 0
function shortLabel(label: string) {
  const parts = label.split('-')
  if (parts.length === 3) return `${Number(parts[1])}/${Number(parts[2])}`
  return label
}
const overallMax = () => Math.max(1, ...(data.value?.overall.map((t) => t.count) || [1]))
</script>
<template>
  <div>
    <PageHeader eyebrow="Statistics" title="Trending words" description="What people are searching, aggregated into terms over time. Individual user queries are never shown — only grouped, masked word counts." />

    <div v-if="loading" class="admin-loading"><LoadingSpinner /></div>
    <template v-else-if="data">
      <section class="admin-card">
        <div class="mini-head">
          <h2>Term trends</h2>
          <div class="word-window-switch" role="tablist" aria-label="Trend window">
            <button type="button" :class="{ active: window === 'daily' }" @click="switchWindow('daily')">Last 7 days</button>
            <button type="button" :class="{ active: window === 'weekly' }" @click="switchWindow('weekly')">Last 8 weeks</button>
          </div>
        </div>
        <div v-if="topWords().length" class="word-chart">
          <div v-for="bucket in data.buckets" :key="bucket.label" class="word-chart-col">
            <div class="word-chart-bars">
              <div v-for="(word, i) in topWords()" :key="word.word" class="word-chart-bar-wrap" :title="`${word.word}: ${countFor(bucket.label, word.word)}`">
                <span
                  class="word-chart-bar"
                  :style="{ height: `${Math.max((countFor(bucket.label, word.word) / chartMax()) * 100, countFor(bucket.label, word.word) ? 4 : 2)}%`, background: CHART_COLORS[i % CHART_COLORS.length] }"
                ></span>
              </div>
            </div>
            <span class="word-chart-label">{{ shortLabel(bucket.label) }}</span>
          </div>
        </div>
        <div v-else class="trend-empty">No terms recorded in this window yet.</div>
        <div v-if="topWords().length" class="word-legend">
          <span v-for="(word, i) in topWords()" :key="word.word" class="word-legend-item">
            <i :style="{ background: CHART_COLORS[i % CHART_COLORS.length] }"></i>{{ word.word }}
          </span>
        </div>
      </section>

      <section class="admin-card">
        <div class="mini-head"><h2>Most searched terms</h2><span class="trend-max">{{ window === 'daily' ? 'last 7 days' : 'last 8 weeks' }}</span></div>
        <ul v-if="data.overall.length" class="rank-list rank-list--full">
          <li v-for="(term, i) in data.overall" :key="term.word">
            <span class="rank-index">{{ i + 1 }}</span>
            <span class="rank-bar" :style="{ width: `${(term.count / overallMax()) * 100}%` }"></span>
            <span class="rank-query">{{ term.word }}</span><span class="rank-count">{{ term.count }}</span>
          </li>
        </ul>
        <EmptyState v-else title="No terms yet" message="Searched words will appear here as users ask the assistant." />
      </section>
    </template>
  </div>
</template>

<style scoped>
.word-window-switch { display: inline-flex; gap: 4px; padding: 3px; border: 1px solid var(--color-border); border-radius: 10px; background: var(--color-surface-subtle); }
.word-window-switch button { border: 0; border-radius: 8px; padding: 6px 12px; background: transparent; color: var(--color-muted); font-size: .74rem; font-weight: 700; cursor: pointer; transition: background .14s ease, color .14s ease; }
.word-window-switch button.active { background: var(--color-surface); color: var(--color-primary); box-shadow: 0 2px 8px var(--color-shadow); }
.word-chart { display: flex; align-items: stretch; gap: 10px; height: 210px; padding-top: 10px; }
.word-chart-col { display: grid; flex: 1; grid-template-rows: 1fr auto; gap: 8px; min-width: 0; }
.word-chart-bars { display: flex; align-items: flex-end; justify-content: space-evenly; gap: 4px; height: 100%; border-bottom: 1px solid var(--color-border); }
.word-chart-bar-wrap { display: flex; align-items: flex-end; height: 100%; flex: 1; max-width: 18px; }
.word-chart-bar { display: block; width: 100%; border-radius: 5px 5px 2px 2px; opacity: .92; transition: height .3s ease; }
.word-chart-label { color: var(--color-muted); font-size: .68rem; font-weight: 650; text-align: center; white-space: nowrap; }
.word-legend { display: flex; flex-wrap: wrap; gap: 14px; margin-top: 14px; padding-top: 12px; border-top: 1px solid var(--color-border); }
.word-legend-item { display: inline-flex; align-items: center; gap: 6px; color: var(--color-muted); font-size: .74rem; font-weight: 650; }
.word-legend-item i { width: 9px; height: 9px; border-radius: 3px; }
@media (max-width: 640px) {
  .word-chart { height: 160px; }
  .word-window-switch { margin-top: 10px; }
}
</style>
