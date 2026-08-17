<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import WebResult from './WebResult.vue'
import type { Source } from '../services/api'

const props = withDefaults(
  defineProps<{
    sources: Source[]
    query?: string
    heading?: string
    pageSize?: number
    showCount?: boolean
  }>(),
  {
    query: '',
    heading: 'Read more',
    pageSize: 10,
    showCount: true,
  },
)

const page = ref(1)
const pageCount = computed(() => Math.max(1, Math.ceil(props.sources.length / props.pageSize)))
const pageSources = computed(() => props.sources.slice((page.value - 1) * props.pageSize, page.value * props.pageSize))

// New results reset the pager.
watch(() => props.sources, () => { page.value = 1 })

function shortPages(): number[] {
  const total = pageCount.value
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)
  const current = Math.min(page.value, total)
  const start = Math.max(2, current - 2)
  const end = Math.min(total - 1, current + 2)
  const pages: number[] = [1]
  if (start > 2) pages.push(-1)
  for (let p = start; p <= end; p++) pages.push(p)
  if (end < total - 1) pages.push(-2)
  pages.push(total)
  return pages
}
</script>

<template>
  <section class="web-results" :aria-label="heading">
    <div class="sources-heading web-results-head">
      <h2>{{ heading }}</h2>
      <span v-if="showCount">{{ sources.length }} results</span>
    </div>
    <WebResult v-for="source in pageSources" :key="source.position" :source="source" :query="query" />
    <nav v-if="pageCount > 1" class="web-results-pager" aria-label="Result pages">
      <button type="button" class="pager-btn" :disabled="page === 1" @click="page -= 1">‹ Prev</button>
      <template v-for="p in shortPages()" :key="p">
        <span v-if="p < 0" class="pager-ellipsis" aria-hidden="true">…</span>
        <button v-else type="button" class="pager-btn" :class="{ active: p === page }" @click="page = p">{{ p }}</button>
      </template>
      <button type="button" class="pager-btn" :disabled="page === pageCount" @click="page += 1">Next ›</button>
    </nav>
  </section>
</template>

<style scoped>
.web-results { margin-top: 38px; padding-bottom: 4px; }
.web-results-head { margin-bottom: 6px; }
.web-results-pager { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; margin-top: 18px; }
.pager-btn {
  min-width: 32px;
  height: 32px;
  padding: 0 10px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-surface);
  color: var(--color-muted);
  font-size: .78rem;
  font-weight: 680;
  cursor: pointer;
  transition: border-color .14s ease, color .14s ease, background .14s ease;
}
.pager-btn:hover:not(:disabled) { border-color: var(--color-primary); color: var(--color-primary); }
.pager-btn.active {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--color-primary-contrast);
}
.pager-btn:disabled { opacity: .45; cursor: not-allowed; }
.pager-ellipsis { color: var(--color-muted); font-size: .8rem; padding: 0 2px; }
</style>
