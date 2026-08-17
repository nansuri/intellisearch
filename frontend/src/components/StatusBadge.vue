<script setup lang="ts">
import { computed } from 'vue'
const props = defineProps<{ value: string; options: Record<string, { label: string; tone?: string }> }>()
const tone = computed(() => props.options[props.value]?.tone || 'neutral')
</script>
<template>
  <span class="status-badge" :class="`status-badge--${tone}`">{{ options[value]?.label || value }}</span>
</template>