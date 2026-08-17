<script setup lang="ts">
import { computed } from 'vue'
const props = defineProps<{ name: string; avatarUrl?: string | null }>()
const initials = computed(() => props.name.trim().split(/\s+/).slice(0, 2).map((w) => w[0]?.toUpperCase() || '').join(''))
</script>
<template>
  <img v-if="avatarUrl" class="avatar-img" :src="avatarUrl" :alt="name" />
  <span v-else class="avatar-initials">{{ initials || '?' }}</span>
</template>