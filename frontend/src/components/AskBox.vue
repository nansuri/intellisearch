<script setup lang="ts">
import { ref } from 'vue'
import BaseButton from './BaseButton.vue'

const props = withDefaults(defineProps<{ variant?: 'default' | 'google'; compact?: boolean; helperText?: string; suggestions?: string[]; showPrompt?: boolean; placeholder?: string }>(), {
  variant: 'default',
  compact: false,
  helperText: '',
  suggestions: () => [],
  showPrompt: true,
  placeholder: '',
})
const emit = defineEmits<{ submit: [question: string] }>()
const value = ref('')

function submit(question = value.value) {
  const cleanedQuestion = question.trim()
  if (!cleanedQuestion) return
  emit('submit', cleanedQuestion)
  value.value = ''
}
</script>

<template>
  <div class="ask-box" :class="[`ask-box--${props.variant}`, { 'ask-box--compact': props.compact }]">
    <form class="ask-form" @submit.prevent="submit()">
      <svg v-if="props.variant === 'google'" class="ask-icon" viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7" fill="none" stroke="currentColor" stroke-width="2" /><path d="M16.5 16.5L21 21" stroke="currentColor" stroke-width="2" stroke-linecap="round" /></svg>
      <span v-if="props.showPrompt" class="ask-prompt" aria-hidden="true">Ask</span>
      <input v-model="value" aria-label="Ask a question" :placeholder="props.placeholder || (props.variant === 'google' ? 'Ask a question, explore an idea…' : props.compact ? 'Ask a follow-up' : 'Ask a question, explore an idea…')" />
      <BaseButton type="submit" variant="primary">{{ props.variant === 'google' ? 'Ask' : props.compact ? 'Search' : 'Ask AI' }}</BaseButton>
    </form>
    <p v-if="props.helperText" class="ask-helper">{{ props.helperText }}</p>
    <div v-if="props.suggestions.length" class="prompt-list" aria-label="Suggested questions">
      <button v-for="suggestion in props.suggestions" :key="suggestion" type="button" @click="submit(suggestion)">{{ suggestion }}</button>
    </div>
  </div>
</template>