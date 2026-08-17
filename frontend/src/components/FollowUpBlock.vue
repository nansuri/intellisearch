<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import ErrorBanner from './ErrorBanner.vue'
import FollowUpLoading from './FollowUpLoading.vue'
import CollapsibleAnswer from './CollapsibleAnswer.vue'
import ImageGrid from './ImageGrid.vue'
import type { ImageItem, Source } from '../services/api'

export type FollowUpEntry = {
  id: number
  question: string
  answer: string
  sources: Source[]
  images: ImageItem[]
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
    <div class="follow-up-label"><span class="follow-up-badge">Follow-up {{ index + 1 }}</span></div>
    <p class="follow-up-question" :class="{ 'follow-up-question--pulse': entry.loading }">{{ entry.question }}</p>

    <ErrorBanner v-if="entry.error" :message="entry.error" />

    <FollowUpLoading v-else-if="entry.loading" />

    <template v-else>
      <CollapsibleAnswer
        :label="searchOnly ? 'Summary from top results' : 'AI overview'"
        :answer="entry.answer"
        :sources="entry.sources"
        :collapsed="entry.collapsed"
        :highlighted="entry.highlighted"
        :empty-label="searchOnly ? 'No web results to summarize.' : undefined"
        compact
        class="follow-up-answer"
        @update:collapsed="emit('update:collapsed', $event)"
      />
      <ImageGrid v-if="entry.images.length" :images="entry.images" class="follow-up-images" />
    </template>
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
.follow-up-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 30%, var(--color-border));
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-primary) 7%, var(--color-surface));
  color: var(--color-primary);
  font-size: .68rem;
  font-weight: 760;
  letter-spacing: .06em;
  text-transform: uppercase;
}
.follow-up-images { margin-top: 8px; }
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
