<script setup lang="ts">
import { useToastStore } from '../stores/toast'
const toast = useToastStore()
</script>
<template>
  <div class="toast-host" aria-live="polite">
    <TransitionGroup name="toast">
      <div v-for="t in toast.toasts" :key="t.id" class="toast" :class="`toast--${t.kind}`" @click="toast.dismiss(t.id)">
        <span class="toast-dot"></span>{{ t.message }}
      </div>
    </TransitionGroup>
  </div>
</template>