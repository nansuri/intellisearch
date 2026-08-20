<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import FollowUpBlock, { type FollowUpEntry } from './FollowUpBlock.vue'
import { suggestFollowUps } from '../services/api'

// "AI Summary" tab hosts the follow-up conversation only. The initial AI
// overview lives in the All tab (CollapsibleAnswer), so a summary never
// appears twice. The parent owns the follow-up API call; this component only
// presents the conversation and emits user intent.
//
// Follow-up suggestions are trigger-driven to save tokens: they are composed
// (a small LLM call) only when the user taps "Suggest follow-up questions" —
// never automatically after an answer.
const props = withDefaults(
  defineProps<{
    sessionId?: string
    thread?: FollowUpEntry[]
    activeFollowUpId?: number | null
    canFollowUp?: boolean
    searchOnly?: boolean
  }>(),
  {
    sessionId: '',
    thread: () => [],
    activeFollowUpId: null,
    canFollowUp: true,
    searchOnly: false,
  }
)

const emit = defineEmits<{
  'update:collapsed-followup': [id: number, value: boolean]
  'follow-up': [question: string]
}>()

const question = ref('')
const suggestions = ref<string[]>([])
const suggestionsLoading = ref(false)
const suggestionsError = ref(false)

const busy = computed(() => props.thread.some((entry) => entry.loading))
const somethingPending = computed(() => props.thread.some((entry) => entry.loading || entry.highlighted))
const canSend = computed(() => Boolean(props.canFollowUp) && !busy.value && question.value.trim().length > 0)

// A new search (session change) or a new follow-up invalidates whatever was
// suggested for the previous conversation.
watch(() => props.sessionId, resetSuggestions)
watch(() => props.thread.length, resetSuggestions)

function resetSuggestions() {
  suggestions.value = []
  suggestionsError.value = false
}

function send() {
  const q = question.value.trim()
  if (!q || !canSend.value) return
  emit('follow-up', q)
  question.value = ''
}

async function loadSuggestions() {
  if (!props.sessionId || suggestionsLoading.value || busy.value) return
  suggestionsLoading.value = true
  suggestionsError.value = false
  try {
    const result = await suggestFollowUps(props.sessionId)
    suggestions.value = result.suggestions || []
  } catch {
    suggestionsError.value = true
  } finally {
    suggestionsLoading.value = false
  }
}

function pick(entry: string) {
  resetSuggestions()
  emit('follow-up', entry)
}
</script>

<template>
  <section class="ai-summary-tab">
    <div class="follow-up-panel">
      <div class="follow-up-heading">
        <h2 class="follow-up-title">AI Summary</h2>
        <p class="follow-up-subtitle">Continue this conversation — ask a follow-up and each answer is built on what came before.</p>
      </div>

      <form class="follow-up-composer" @submit.prevent="send">
        <svg class="follow-up-composer-icon" viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
          <path d="M12 19V5M6 11l6-6 6 6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        <input
          v-model="question"
          :disabled="busy"
          :placeholder="busy ? 'Researching your follow-up…' : 'Ask a follow-up question…'"
          aria-label="Follow-up question"
        />
        <button class="follow-up-composer-send" type="submit" :disabled="!canSend" aria-label="Send follow-up">
          {{ busy ? 'Thinking…' : 'Send' }}
        </button>
      </form>
      <p v-if="!canFollowUp" class="follow-up-hint">Start a new search in the box above to ask a different question.</p>

      <section v-if="sessionId" class="suggestions-box" aria-label="Suggested follow-up questions">
        <template v-if="suggestions.length">
          <div class="suggestions-head">
            <span class="suggestions-label">Try a follow-up</span>
            <button
              class="suggestions-refresh"
              type="button"
              title="Compose new suggestions"
              :disabled="suggestionsLoading || busy"
              @click="loadSuggestions"
            >↻</button>
          </div>
          <div class="suggestions-chips">
            <button
              v-for="entry in suggestions"
              :key="entry"
              type="button"
              class="suggestion-chip"
              :disabled="busy"
              @click="pick(entry)"
            >{{ entry }}</button>
          </div>
        </template>
        <button
          v-else
          type="button"
          class="suggest-trigger"
          :disabled="suggestionsLoading || busy"
          @click="loadSuggestions"
        >
          <svg class="suggest-trigger-icon" viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
            <path d="M12 2.4l1.9 5.2 5.2 1.9-5.2 1.9L12 16.6l-1.9-5.2-5.2-1.9 5.2-1.9z" fill="currentColor" />
            <path d="M19.5 14.5l.8 2.2 2.2.8-2.2.8-.8 2.2-.8-2.2-2.2-.8 2.2-.8z" fill="currentColor" />
          </svg>
          {{ suggestionsLoading ? 'Composing suggestions…' : 'Suggest follow-up questions' }}
        </button>
        <p v-if="suggestionsError" class="suggestions-note">Couldn't compose suggestions right now — try again.</p>
      </section>

      <section v-if="thread.length" class="follow-up-thread">
        <div v-if="somethingPending" class="follow-up-thread-hint">
          <span class="live-dot" />
          New answer in progress — previous answers are summarized above.
        </div>
        <FollowUpBlock
          v-for="(entry, index) in thread"
          :key="entry.id"
          :entry="entry"
          :index="index"
          :active="entry.id === activeFollowUpId"
          :search-only="searchOnly"
          @update:collapsed="emit('update:collapsed-followup', entry.id, $event)"
        />
      </section>
    </div>
  </section>
</template>

<style scoped>
.ai-summary-tab {
  min-width: 0;
  padding-top: 6px;
}

.follow-up-panel { margin-top: 18px; }
.follow-up-heading { margin-bottom: 14px; }
.follow-up-title {
  margin: 0;
  font-size: 1.15rem;
  letter-spacing: -.025em;
}
.follow-up-subtitle {
  margin: 4px 0 0;
  color: var(--color-muted);
  font-size: .8rem;
}

.follow-up-composer {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 5px 5px 5px 16px;
  border: 1px solid var(--color-border);
  border-radius: 16px;
  background: var(--color-surface);
  box-shadow: var(--shadow-search);
  transition: border-color .18s ease, box-shadow .18s ease;
}
.follow-up-composer:focus-within {
  border-color: color-mix(in srgb, var(--color-primary) 50%, var(--color-border));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-focus) 28%, transparent), var(--shadow-search);
}
.follow-up-composer-icon { flex: 0 0 auto; color: var(--color-muted); }
.follow-up-composer input {
  min-width: 0;
  flex: 1;
  padding: 10px 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--color-text);
  font-size: .92rem;
}
.follow-up-composer input::placeholder { color: var(--color-muted); opacity: 1; }
.follow-up-composer input:disabled { cursor: not-allowed; }
.follow-up-composer-send {
  flex: 0 0 auto;
  min-height: 38px;
  padding: 0 18px;
  border: 0;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-primary-contrast);
  font-size: .8rem;
  font-weight: 720;
  cursor: pointer;
  transition: background .16s ease, opacity .16s ease;
}
.follow-up-composer-send:hover:not(:disabled) { background: var(--color-primary-hover); }
.follow-up-composer-send:disabled { opacity: .5; cursor: not-allowed; }
.follow-up-hint {
  margin: 10px 2px 0;
  color: var(--color-muted);
  font-size: .76rem;
}

.follow-up-thread {
  margin-top: 36px;
  padding-top: 6px;
  border-top: 1px solid var(--color-border);
}
.follow-up-thread-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  padding: 10px 14px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 24%, var(--color-border));
  border-radius: 12px;
  background: color-mix(in srgb, var(--color-primary) 5%, var(--color-surface));
  color: var(--color-muted);
  font-size: .78rem;
}

.suggestions-box {
  display: grid;
  gap: 10px;
  margin-top: 18px;
  padding: 14px 16px;
  border: 1px dashed var(--color-border);
  border-radius: 14px;
  background: var(--color-surface-subtle);
}
.suggestions-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.suggestions-label {
  color: var(--color-muted);
  font-size: .72rem;
  font-weight: 700;
  letter-spacing: .08em;
  text-transform: uppercase;
}
.suggestions-refresh {
  padding: 2px 6px;
  border: 0;
  background: transparent;
  color: var(--color-muted);
  cursor: pointer;
  font-size: .85rem;
  transition: color .14s ease, transform .3s ease;
}
.suggestions-refresh:hover:not(:disabled) { color: var(--color-primary); }
.suggestions-refresh:disabled { opacity: .4; cursor: not-allowed; }
.suggestions-refresh:not(:disabled):active { transform: rotate(180deg); }
.suggestions-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.suggestion-chip {
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-surface);
  color: var(--color-text);
  font-size: .8rem;
  line-height: 1.35;
  text-align: left;
  cursor: pointer;
  transition: border-color .14s ease, color .14s ease, background .14s ease;
}
.suggestion-chip:hover:not(:disabled) {
  border-color: var(--color-primary);
  color: var(--color-primary);
  background: color-mix(in srgb, var(--color-primary) 6%, var(--color-surface));
}
.suggestion-chip:disabled { opacity: .55; cursor: not-allowed; }
.suggest-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  width: fit-content;
  padding: 8px 14px;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-surface);
  color: var(--color-primary);
  font-size: .78rem;
  font-weight: 680;
  cursor: pointer;
  transition: border-color .14s ease, color .14s ease;
}
.suggest-trigger:hover:not(:disabled) { border-color: var(--color-primary); }
.suggest-trigger:disabled { color: var(--color-muted); cursor: not-allowed; }
.suggest-trigger-icon { flex: 0 0 auto; }
.suggestions-note {
  margin: 0;
  color: var(--color-muted);
  font-size: .74rem;
}

@media (max-width: 400px) {
  .follow-up-composer { padding-left: 12px; }
}
</style>