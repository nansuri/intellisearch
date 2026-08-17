<script setup lang="ts">
defineProps<{ page: number; totalPages: number; total: number }>()
const emit = defineEmits<{ (e: 'change', page: number): void }>()
const pages = (total: number) => Array.from({ length: total }, (_, i) => i + 1)
</script>
<template>
  <nav v-if="totalPages > 1" class="pagination" aria-label="Pagination">
    <button class="base-button button-secondary" :disabled="page <= 1" @click="emit('change', page - 1)">Previous</button>
    <button v-for="p in pages(totalPages)" :key="p" class="page-dot" :class="{ active: p === page }" :aria-current="p === page" @click="emit('change', p)">{{ p }}</button>
    <button class="base-button button-secondary" :disabled="page >= totalPages" @click="emit('change', page + 1)">Next</button>
    <span class="pagination-total">{{ total }} result{{ total === 1 ? '' : 's' }}</span>
  </nav>
</template>