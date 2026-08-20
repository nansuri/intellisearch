<script setup lang="ts">
import { computed } from 'vue'
import MarkdownView from './MarkdownView.vue'
import MapCard from './MapCard.vue'
import WebResultList from './WebResultList.vue'
import { answerPreview } from '../utils/answerSummary'
import type { MapPoint, Source } from '../services/api'

const props = withDefaults(defineProps<{
  label: string
  answer: string
  sources?: Source[]
  query?: string
  collapsed?: boolean
  highlighted?: boolean
  showSources?: boolean
  compact?: boolean
  emptyLabel?: string
  mapCenter?: MapPoint | null
  mapMarkers?: MapPoint[]
}>(), {
  sources: () => [],
  query: '',
  collapsed: false,
  highlighted: false,
  showSources: true,
  compact: false,
  emptyLabel: 'No answer yet. Try asking again.',
  mapCenter: null,
  mapMarkers: () => [],
})

const emit = defineEmits<{ 'update:collapsed': [value: boolean] }>()

const preview = computed(() => answerPreview(props.answer))
const canCollapse = computed(() => Boolean(props.answer))

function expand() { emit('update:collapsed', false) }
function collapse() { emit('update:collapsed', true) }
</script>

<template>
  <article
    class="collapsible-answer"
    :class="{
      'collapsible-answer--collapsed': collapsed && canCollapse,
      'collapsible-answer--highlight': highlighted,
      'collapsible-answer--compact': compact,
    }"
  >
    <button
      v-if="collapsed && canCollapse"
      type="button"
      class="collapse-bar"
      :aria-expanded="false"
      @click="expand"
    >
      <span class="collapse-bar-label">
        <svg class="collapse-bar-icon" viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
          <path d="M12 2.4l1.9 5.2 5.2 1.9-5.2 1.9L12 16.6l-1.9-5.2-5.2-1.9 5.2-1.9z" fill="currentColor" />
          <path d="M19.5 14.5l.8 2.2 2.2.8-2.2.8-.8 2.2-.8-2.2-2.2-.8 2.2-.8z" fill="currentColor" />
        </svg>
        {{ label }}
      </span>
      <span class="collapse-bar-preview">{{ preview }}</span>
      <span v-if="sources.length" class="collapse-bar-count">{{ sources.length }} result{{ sources.length === 1 ? '' : 's' }}</span>
      <span class="collapse-bar-action">
        Show summary
        <svg class="collapse-bar-arrow" viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
          <path d="M9 5l7 7-7 7" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </span>
    </button>

    <template v-else>
      <div class="collapsible-answer-head">
        <div class="section-label">{{ label }}</div>
        <button v-if="canCollapse" type="button" class="collapse-toggle" @click="collapse">Hide summary</button>
      </div>
      <MarkdownView v-if="answer" :content="answer" />
      <p v-else>{{ emptyLabel }}</p>

      <MapCard v-if="mapCenter || mapMarkers.length" :center="mapCenter" :markers="mapMarkers" />

      <WebResultList
        v-if="showSources && sources.length"
        :sources="sources"
        :query="query"
        heading="Read more"
        class="sources--nested"
      />
    </template>
  </article>
</template>

<style scoped>
.collapsible-answer {
  display: grid;
  gap: 16px;
  /* min-width: 0 lets the grid shrink below its content width so wide
     markdown (tables, long URLs, code) stays inside the parent container
     on narrow / foldable screens. overflow: hidden clips anything that still
     manages to escape (wide tables handle their own scroll). */
  min-width: 0;
  overflow: hidden;
  margin-top: 38px;
  padding: 0 0 28px;
  border-bottom: 1px solid var(--color-border);
  transition: transform .45s cubic-bezier(.22, .9, .28, 1), box-shadow .45s ease;
}
.collapsible-answer--collapsed {
  margin-top: 18px;
  padding-bottom: 0;
  border-bottom: 0;
}
.collapsible-answer--highlight {
  animation: answer-spotlight 2.4s ease-out;
}
.collapsible-answer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.collapse-toggle {
  padding: 4px 10px;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: transparent;
  color: var(--color-muted);
  cursor: pointer;
  font-size: .72rem;
  font-weight: 650;
  transition: border-color .14s ease, color .14s ease;
}
.collapse-toggle:hover { border-color: var(--color-primary); color: var(--color-primary); }
.collapse-bar {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  min-width: 0;
  padding: 14px 16px;
  border: 1px solid var(--color-border);
  border-radius: 14px;
  background: var(--color-surface);
  box-shadow: 0 2px 10px var(--color-shadow);
  color: inherit;
  cursor: pointer;
  text-align: left;
  transition: border-color .16s ease, transform .16s ease, box-shadow .16s ease;
}
.collapse-bar:hover {
  border-color: color-mix(in srgb, var(--color-primary) 40%, var(--color-border));
  transform: translateY(-1px);
  box-shadow: 0 6px 16px var(--color-shadow);
}
.collapse-bar-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 0 0 auto;
  color: var(--color-primary);
  font-size: .66rem;
  font-weight: 760;
  letter-spacing: .09em;
  text-transform: uppercase;
}
.collapse-bar-icon { flex: 0 0 auto; }
.collapse-bar-preview {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--color-muted);
  font-size: .82rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.collapse-bar-count {
  flex: 0 0 auto;
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--color-surface-subtle);
  color: var(--color-muted);
  font-size: .68rem;
  font-weight: 680;
  white-space: nowrap;
}
.collapse-bar-action {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  flex: 0 0 auto;
  color: var(--color-primary);
  font-size: .76rem;
  font-weight: 700;
  white-space: nowrap;
  transition: color .16s ease;
}
.collapse-bar:hover .collapse-bar-action { color: var(--color-primary-hover); }
.collapse-bar-arrow { transition: transform .16s ease; }
.collapse-bar:hover .collapse-bar-arrow { transform: translateX(3px); }
.sources--nested { margin-top: 0; }
:deep(.web-results) { margin-top: 6px; }
.collapsible-answer--compact { margin-top: 0; padding-bottom: 20px; }
.collapsible-answer--compact.collapsible-answer--collapsed { margin-top: 0; }
@media (max-width: 560px) {
  .collapse-bar-preview { display: none; }
}
@keyframes answer-spotlight {
  0%, 12% { box-shadow: none; transform: scale(1); }
  22% {
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-primary) 42%, transparent),
      0 14px 42px color-mix(in srgb, var(--color-primary) 16%, transparent);
    transform: scale(1.008);
  }
  100% { box-shadow: none; transform: scale(1); }
}
</style>