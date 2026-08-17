<script setup lang="ts">
defineProps<{ eyebrow?: string; title: string; description?: string }>()
</script>
<template>
  <header class="admin-page-head">
    <div>
      <div v-if="eyebrow" class="section-label">{{ eyebrow }}</div>
      <h1>{{ title }}</h1>
      <p v-if="description">{{ description }}</p>
    </div>
    <div class="admin-page-head-actions"><slot /></div>
  </header>
</template>