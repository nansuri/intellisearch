<script setup lang="ts">
import BaseButton from './BaseButton.vue'
defineProps<{ message: string }>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <div class="error-banner" role="alert">
    <span class="error-banner-copy">{{ message }}</span>
    <BaseButton variant="secondary" @click="emit('retry')">Try again</BaseButton>
  </div>
</template>
