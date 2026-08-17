<script setup lang="ts">
import { ref } from 'vue'
import BaseButton from './BaseButton.vue'

const emit = defineEmits<{ submit: [url: string] }>()
const value = ref('')

function submit() {
  const cleaned = value.value.trim()
  if (!cleaned) return
  emit('submit', cleaned)
  value.value = ''
}
</script>

<template>
  <div class="url-ask-box">
    <form class="url-ask-form" @submit.prevent="submit">
      <span class="url-ask-prompt" aria-hidden="true">Read a URL</span>
      <input v-model="value" aria-label="Submit a URL to read" placeholder="https://example.com/article" type="url" />
      <BaseButton type="submit" variant="primary">Read</BaseButton>
    </form>
  </div>
</template>
