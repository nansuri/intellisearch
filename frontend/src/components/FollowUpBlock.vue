<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import ErrorBanner from './ErrorBanner.vue'
import FollowUpLoading from './FollowUpLoading.vue'
import CollapsibleAnswer from './CollapsibleAnswer.vue'
import type { Source } from '../services/api'

export type FollowUpEntry = {
  id: number
  question: string
  answer: string
  sources: Source[]
  error: string | null
  loading: boolean
  collapsed: boolean
  highlighted: boolean
}

const props = withDefaults(defineProps<{ entry: FollowUpEntry; index: number; active: boolean; searchOnly?: boolean }>(), { searchOnly: false })
const emit = defineEmits<{ 'update:collapsed': [value: boolean] }>()
const root = ref<HTMLElement | null>(null)

function scrollToBlock(behavior: ScrollBehavior = 'smooth') {
  root.value?.scrollIntoView({ behavior, block: 'start' })
}

onMounted(() => {
  if (props.active) scrollToBlock('smooth')
})

watch(() => props.entry.loading, async (loading, wasLoading) => {
  if (loading) {
    await nextTick()
    scrollToBlock('smooth')
  }
  if (wasLoading && !loading && !props.entry.error) {
    await nextTick()
    scrollToBlock('smooth')
  }
})

defineExpose({ scrollToBlock })
</script>

<template>
  <div
    ref="root"
    class="follow-up-item"
    :class="{
      'follow-up-item--active': active,
      'follow-up-item--loading': entry.loading,
      'follow-up-item--highlight': entry.highlighted,
    }"
  >
    <div class="follow-up-label">Follow-up {{ index + 1 }}</div>
    <p class="follow-up-question" :class="{ 'follow-up-question--pulse': entry.loading }">{{ entry.question }}</p>

    <ErrorBanner v-if="entry.error" :message="entry.error" />

    <FollowUpLoading v-else-if="entry.loading" />

    <CollapsibleAnswer
      v-else
      :label="searchOnly ? 'Web results' : 'AI overview'"
      :answer="entry.answer"
      :sources="entry.sources"
      :collapsed="entry.collapsed"
      :highlighted="entry.highlighted"
      :empty-label="searchOnly ? 'Raw web results — no AI summary.' : undefined"
      compact
      class="follow-up-answer"
      @update:collapsed="emit('update:collapsed', $event)"
    />
  </div>
</template>

<style scoped>
.follow-up-item {
  margin-top: 30px;
  scroll-margin-top: 92px;
  transition: transform .35s ease;
}
.follow-up-item--active { scroll-margin-top: 92px; }
.follow-up-item--loading .follow-up-question { color: var(--color-primary); }
.follow-up-item--highlight {
  animation: follow-up-nudge .55s cubic-bezier(.22, .9, .28, 1);
}
.follow-up-question--pulse {
  animation: follow-up-question-pulse 1.6s ease-in-out infinite;
}
.follow-up-answer { margin-top: 0; }
@keyframes follow-up-nudge {
  0% { transform: translateY(6px); opacity: .72; }
  100% { transform: translateY(0); opacity: 1; }
}
@keyframes follow-up-question-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: .72; }
}
</style>
