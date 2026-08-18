<script setup lang="ts">
import { computed, ref } from 'vue'
import BaseButton from './BaseButton.vue'
import RadioSwitch from './RadioSwitch.vue'
import type { AskMode } from '../services/api'

const props = withDefaults(defineProps<{ variant?: 'default' | 'google'; compact?: boolean; helperText?: string; suggestions?: string[]; showPrompt?: boolean; placeholder?: string; mode?: AskMode; modeToggle?: boolean; hasSession?: boolean; followUp?: boolean }>(), {
  variant: 'default',
  compact: false,
  helperText: '',
  suggestions: () => [],
  showPrompt: true,
  placeholder: '',
  mode: 'enhanced',
  modeToggle: false,
  hasSession: false,
  followUp: false,
})
const emit = defineEmits<{ submit: [question: string, mode: AskMode]; 'update:mode': [mode: AskMode]; 'update:followUp': [value: boolean] }>()
const value = ref('')

const activeMode = computed<AskMode>(() => props.mode || 'enhanced')
const submitLabel = computed(() => (activeMode.value === 'search' ? 'Search' : props.variant === 'google' ? 'Ask' : props.compact ? 'Search' : 'Ask AI'))

const modeOptions = [
  { value: 'enhanced', label: 'Ask AI', title: 'AI answer + cited sources' },
  { value: 'search', label: 'Search', title: 'Summary from results · no AI used' },
]
const sendMode = computed(() => (props.followUp ? 'followup' : 'newsearch'))
const sendModeOptions = [
  { value: 'followup', label: 'Follow-up', title: 'Continues the current conversation with a follow-up answer' },
  { value: 'newsearch', label: 'New search', title: 'Starts a fresh search with the typed question' },
]

function submit(question = value.value) {
  const cleanedQuestion = question.trim()
  if (!cleanedQuestion) return
  emit('submit', cleanedQuestion, activeMode.value)
  value.value = ''
}
function setMode(mode: string) {
  if (mode !== 'enhanced' && mode !== 'search') return
  emit('update:mode', mode)
}
function setSendMode(mode: string) {
  if (mode !== 'followup' && mode !== 'newsearch') return
  emit('update:followUp', mode === 'followup')
}
</script>

<template>
  <div class="ask-box" :class="[`ask-box--${props.variant}`, { 'ask-box--compact': props.compact }]">
    <form class="ask-form" @submit.prevent="submit()">
      <svg v-if="props.variant === 'google'" class="ask-icon" viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7" fill="none" stroke="currentColor" stroke-width="2" /><path d="M16.5 16.5L21 21" stroke="currentColor" stroke-width="2" stroke-linecap="round" /></svg>
      <span v-if="props.showPrompt" class="ask-prompt" aria-hidden="true">Ask</span>
      <input v-model="value" aria-label="Ask a question" :placeholder="props.placeholder || (props.variant === 'google' ? 'Ask a question, explore an idea…' : props.compact ? 'Ask a follow-up' : 'Ask a question, explore an idea…')" />
      <RadioSwitch
        v-if="props.hasSession"
        class="ask-send-switch"
        variant="pill"
        size="sm"
        :options="sendModeOptions"
        :model-value="sendMode"
        aria-label="Send as follow-up or new search"
        @update:model-value="setSendMode"
      />
      <BaseButton type="submit" variant="primary">{{ submitLabel }}</BaseButton>
    </form>

    <div v-if="props.modeToggle" class="ask-mode">
      <RadioSwitch
        variant="segment"
        :options="modeOptions"
        :model-value="activeMode"
        aria-label="Ask mode"
        @update:model-value="setMode"
      />
      <span class="ask-mode-hint">{{ activeMode === 'search' ? 'Summary from results · no AI used' : 'AI answer + cited sources' }}</span>
    </div>

    <p v-if="props.helperText" class="ask-helper">{{ props.helperText }}</p>
    <div v-if="props.suggestions.length" class="prompt-list" aria-label="Suggested questions">
      <button v-for="suggestion in props.suggestions" :key="suggestion" type="button" @click="submit(suggestion)">{{ suggestion }}</button>
    </div>
  </div>
</template>

<style scoped>
.ask-mode {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 14px;
}
.ask-mode-hint { margin-left: 6px; padding-left: 8px; border-left: 1px solid var(--color-border); color: var(--color-muted); font-size: .7rem; }
.ask-send-switch :deep(.radio-switch) { border: 0; border-left: 1px solid var(--color-border); border-radius: 0; background: transparent; }
.ask-send-switch :deep(.radio-switch--pill button) { min-height: 34px; }
@media (max-width: 520px) { .ask-mode-hint { display: none; } }
@media (max-width: 380px) { .ask-send-switch :deep(.radio-switch button) { font-size: 0; gap: 4px; } }
</style>
