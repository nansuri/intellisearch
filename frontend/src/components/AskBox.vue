<script setup lang="ts">
import { computed, ref } from 'vue'
import BaseButton from './BaseButton.vue'
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

function submit(question = value.value) {
  const cleanedQuestion = question.trim()
  if (!cleanedQuestion) return
  emit('submit', cleanedQuestion, activeMode.value)
  value.value = ''
}
function setMode(mode: AskMode) { emit('update:mode', mode) }
</script>

<template>
  <div class="ask-box" :class="[`ask-box--${props.variant}`, { 'ask-box--compact': props.compact }]">
    <form class="ask-form" @submit.prevent="submit()">
      <svg v-if="props.variant === 'google'" class="ask-icon" viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7" fill="none" stroke="currentColor" stroke-width="2" /><path d="M16.5 16.5L21 21" stroke="currentColor" stroke-width="2" stroke-linecap="round" /></svg>
      <span v-if="props.showPrompt" class="ask-prompt" aria-hidden="true">Ask</span>
      <input v-model="value" aria-label="Ask a question" :placeholder="props.placeholder || (props.variant === 'google' ? 'Ask a question, explore an idea…' : props.compact ? 'Ask a follow-up' : 'Ask a question, explore an idea…')" />
      <button
        v-if="props.hasSession"
        type="button"
        class="ask-send-mode"
        :class="{ 'ask-send-mode--active': props.followUp }"
        :title="props.followUp ? 'Sending as a follow-up — click to start a new search' : 'Will start a new search — click to follow up instead'"
        @click="emit('update:followUp', !props.followUp)"
      >
        <span class="ask-send-mode-dot" />
        {{ props.followUp ? 'Follow-up' : 'New search' }}
      </button>
      <BaseButton type="submit" variant="primary">{{ submitLabel }}</BaseButton>
    </form>

    <div v-if="props.modeToggle" class="ask-mode" role="radiogroup" aria-label="Ask mode">
      <button
        type="button"
        role="radio"
        :aria-checked="activeMode === 'enhanced'"
        :class="{ active: activeMode === 'enhanced' }"
        @click="setMode('enhanced')"
      >
        <span class="ask-mode-dot" />
        Ask AI
      </button>
      <button
        type="button"
        role="radio"
        :aria-checked="activeMode === 'search'"
        :class="{ active: activeMode === 'search' }"
        @click="setMode('search')"
      >
        <span class="ask-mode-dot" />
        Search
      </button>
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
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 14px;
  padding: 4px;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: var(--color-surface-subtle);
}
.ask-mode button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 32px;
  padding: 0 12px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: var(--color-muted);
  font-size: .78rem;
  font-weight: 700;
  cursor: pointer;
  transition: color .16s ease, background .16s ease, box-shadow .16s ease;
}
.ask-mode button.active {
  color: var(--color-text);
  background: var(--color-surface);
  box-shadow: 0 3px 10px var(--color-shadow);
}
.ask-mode-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--color-muted); transition: background .16s ease; }
.ask-mode button.active .ask-mode-dot { background: var(--color-primary); }
.ask-mode-hint { margin-left: 6px; padding-left: 8px; border-left: 1px solid var(--color-border); color: var(--color-muted); font-size: .7rem; }
.ask-send-mode {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 0 0 auto;
  min-height: 38px;
  padding: 0 13px;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-surface-subtle);
  color: var(--color-muted);
  font-size: .74rem;
  font-weight: 720;
  cursor: pointer;
  white-space: nowrap;
  transition: border-color .16s ease, color .16s ease, background .16s ease;
}
.ask-send-mode:hover { border-color: color-mix(in srgb, var(--color-primary) 42%, var(--color-border)); }
.ask-send-mode--active {
  border-color: color-mix(in srgb, var(--color-primary) 36%, var(--color-border));
  background: color-mix(in srgb, var(--color-primary) 9%, var(--color-surface));
  color: var(--color-primary);
}
.ask-send-mode-dot { width: 7px; height: 7px; border-radius: 50%; background: currentColor; }
@media (max-width: 520px) { .ask-send-mode { min-height: 34px; padding: 0 10px; font-size: .7rem; } }
@media (max-width: 520px) { .ask-mode-hint { display: none; } }
</style>
