<script setup lang="ts">
defineProps<{ icon?: string; title: string; message?: string }>()
</script>
<template>
  <div class="empty-state">
    <span v-if="icon" class="empty-icon">{{ icon }}</span>
    <h3>{{ title }}</h3>
    <p v-if="message">{{ message }}</p>
    <slot />
  </div>
</template>