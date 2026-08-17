<script setup lang="ts">
import { computed } from 'vue'
import type { TrendPoint } from '../services/api'

const props = defineProps<{ title: string; points: TrendPoint[]; emptyMessage?: string }>()
const max = computed(() => Math.max(1, ...props.points.map((p) => p.count)))
const maxCount = computed(() => props.points.reduce((acc, p) => Math.max(acc, p.count), 0))
function shortLabel(label: string) {
  const date = label.split('-')
  if (date.length === 3) return `${Number(date[1])}/${Number(date[2])}`
  return label.replace('-W', ' W')
}
</script>
<template>
  <div class="admin-card trend-card">
    <div class="mini-head"><h2>{{ title }}</h2><span v-if="maxCount > 0" class="trend-max">peak {{ maxCount }}</span></div>
    <div v-if="points.length && maxCount > 0" class="trend-chart">
      <div v-for="p in points" :key="p.label" class="trend-bar-col">
        <span class="trend-bar" :class="{ 'trend-bar--zero': p.count === 0 }" :style="{ height: `${Math.max((p.count / max) * 100, p.count ? 4 : 2)}%` }"></span>
        <span class="trend-bar-label">{{ shortLabel(p.label) }}</span>
      </div>
    </div>
    <div v-else class="trend-empty">{{ emptyMessage || 'No activity recorded in this window.' }}</div>
  </div>
</template>