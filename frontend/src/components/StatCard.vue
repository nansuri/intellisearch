<script setup lang="ts">
defineProps<{ label: string; value: string | number; hint?: string; trend?: string }>()
</script>
<template>
  <div class="stat-card">
    <span class="stat-label">{{ label }}</span>
    <strong class="stat-value">{{ value }}</strong>
    <span v-if="hint" class="stat-hint">{{ hint }}</span>
    <span v-if="trend" class="stat-trend">{{ trend }}</span>
  </div>
</template>