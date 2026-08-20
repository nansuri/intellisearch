<script setup lang="ts">
import { computed, ref } from 'vue'
import CollapsibleAnswer from './CollapsibleAnswer.vue'
import FollowUpBlock, { type FollowUpEntry } from './FollowUpBlock.vue'
import type { MapPoint, Source } from '../services/api'

// Dedicated "AI Summary" tab: the AI overview envelope plus the follow-up
// conversation (composer + thread). The parent owns the data and the follow-up
// API call; this component only presents it and emits user intent.
const props = withDefaults(
  defineProps<{
    answer: string
    sources?: Source[]
    mapCenter?: MapPoint | null
    mapMarkers?: MapPoint[]
    collapsed?: boolean
    thread?: FollowUpEntry[]
    activeFollowUpId?: number | null
    canFollowUp?: boolean
    searchOnly?: boolean
  }>(),
  {
    sources: () => [],
    mapCenter: null,
    mapMarkers: () => [],
    collapsed: false,
    thread: () => [],
    activeFollowUpId: null,
    canFollowUp: true,
    searchOnly: false,
  }
)

const emit = defineEmits<{
  'update:collapsed': [value: boolean]
  'update:collapsed-followup': [id: number, value: boolean]
  'follow-up': [question: string]
}>()

const question = ref('')

const busy = computed(() => props.thread.some((entry) => entry.loading))
const somethingPending = computed(() => props.thread.some((entry) => entry.loading || entry.highlighted))
const canSend = computed(() => Boolean(props.canFollowUp) && !busy.value && question.value.trim().length > 0)

function send() {
  const q = question.value.trim()
  if (!q || !canSend.value) return
  emit('follow-up', q)
  question.value = ''
}
</script>

<template>
  <section class="ai-summary-tab">
    <CollapsibleAnswer
      :label="searchOnly ? 'Summary from top results' : 'AI overview'"
      :answer="answer"
      :sources="sources"
      :collapsed="collapsed"
      :map-center="mapCenter"
      :map-markers="mapMarkers"
      :show-sources="false"
      @update:collapsed="emit('update:collapsed', $event)"
    />

    <div class="follow-up-panel">
      <div class="follow-up-heading">
        <h2 class="follow-up-title">Keep digging</h2>
        <p class="follow-up-subtitle">Ask a follow-up below and the AI keeps building on this conversation.</p>
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

.follow-up-panel { margin-top: 52px; }
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

@media (max-width: 400px) {
  .follow-up-composer { padding-left: 12px; }
}
</style>