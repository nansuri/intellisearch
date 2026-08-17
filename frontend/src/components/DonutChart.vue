<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{ value: number; size?: number; label?: string; tone?: 'primary' | 'danger' }>(), { size: 104, tone: 'primary' })
const clamped = computed(() => Math.min(100, Math.max(0, props.value)))
const R = 9.25 // circle radius in the 24x24 viewBox
const CIRC = 2 * Math.PI * R
const dash = computed(() => `${(clamped.value / 100) * CIRC} ${CIRC}`)
</script>

<template>
  <div class="donut-wrap" :style="{ width: `${size}px`, height: `${size}px` }" role="img" :aria-label="label || `${Math.round(clamped)}%`">
    <svg :width="size" :height="size" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <circle class="donut-track" cx="12" cy="12" :r="R" />
      <circle
        class="donut-fill"
        :class="`donut-fill--${tone}`"
        cx="12"
        cy="12"
        :r="R"
        :stroke-dasharray="dash"
        transform="rotate(-90 12 12)"
      />
    </svg>
    <span class="donut-center">{{ label ?? `${Math.round(clamped)}%` }}</span>
  </div>
</template>

<style scoped>
.donut-wrap { position: relative; display: grid; place-items: center; flex: 0 0 auto; }
.donut-wrap svg { display: block; }
.donut-track { stroke: var(--color-surface-subtle); stroke-width: 2.3; }
.donut-fill { stroke: var(--color-primary); stroke-width: 2.3; stroke-linecap: round; transition: stroke-dasharray .5s ease; }
.donut-fill--danger { stroke: var(--color-danger); }
.donut-center { position: absolute; inset: 0; display: grid; place-items: center; font-size: .95rem; font-weight: 750; letter-spacing: -.02em; font-variant-numeric: tabular-nums; }
</style>
