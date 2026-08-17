<script setup lang="ts">
defineProps<{ label?: string; hint?: string; error?: string }>()
</script>
<template>
  <label class="form-field">
    <span v-if="label" class="form-label">{{ label }}</span>
    <slot />
    <span v-if="hint && !error" class="form-hint">{{ hint }}</span>
    <span v-if="error" class="form-error">{{ error }}</span>
  </label>
</template>