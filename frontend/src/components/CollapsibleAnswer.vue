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
  actionLabel?: string
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
  actionLabel: 'Read full summary',
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
      class="collapsed-chip"
      :aria-expanded="false"
      @click="expand"
    >
      <span class="collapsed-chip-head">
        <span class="collapsed-chip-label">
          <svg class="collapsed-chip-icon" viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
            <path d="M12 2.4l1.9 5.2 5.2 1.9-5.2 1.9L12 16.6l-1.9-5.2-5.2-1.9 5.2-1.9z" fill="currentColor" />
            <path d="M19.5 14.5l.8 2.2 2.2.8-2.2.8-.8 2.2-.8-2.2-2.2-.8 2.2-.8z" fill="currentColor" />
          </svg>
          {{ label }}
        </span>
        <span v-if="sources.length" class="collapsed-chip-count">{{ sources.length }} result{{ sources.length === 1 ? '' : 's' }}</span>
      </span>
      <p class="collapsed-chip-preview">{{ preview }}</p>
      <span class="collapsed-chip-meta">
        <span class="collapsed-chip-hint">Just a taste — dive in for the full picture</span>
        <span class="collapsed-chip-action">
          {{ actionLabel }}
          <svg class="collapsed-chip-arrow" viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
            <path d="M9 5l7 7-7 7" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </span>
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
.collapsed-chip {
  position: relative;
  display: grid;
  gap: 10px;
  width: 100%;
  padding: 16px 18px 14px 22px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 30%, var(--color-border));
  border-radius: 16px;
  background: linear-gradient(180deg, color-mix(in srgb, var(--color-primary) 6%, var(--color-surface)), var(--color-surface));
  box-shadow: var(--shadow-search);
  color: inherit;
  cursor: pointer;
  text-align: left;
  transition: border-color .16s ease, background .16s ease, transform .16s ease, box-shadow .16s ease;
}
.collapsed-chip::before {
  content: '';
  position: absolute;
  top: 12px;
  bottom: 12px;
  left: 0;
  width: 4px;
  border-radius: 0 4px 4px 0;
  background: linear-gradient(180deg, var(--color-primary), var(--color-primary-hover));
}
.collapsed-chip:hover {
  border-color: color-mix(in srgb, var(--color-primary) 55%, var(--color-border));
  background: linear-gradient(180deg, color-mix(in srgb, var(--color-primary) 9%, var(--color-surface)), var(--color-surface));
  transform: translateY(-1px);
  box-shadow: 0 12px 32px var(--color-shadow);
}
.collapsed-chip-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.collapsed-chip-label {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: var(--color-primary);
  font-size: .68rem;
  font-weight: 760;
  letter-spacing: .08em;
  text-transform: uppercase;
}
.collapsed-chip-icon { flex: 0 0 auto; }
.collapsed-chip-count {
  flex: 0 0 auto;
  padding: 4px 10px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 24%, var(--color-border));
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-primary) 8%, var(--color-surface));
  color: var(--color-primary);
  font-size: .7rem;
  font-weight: 680;
  white-space: nowrap;
}
.collapsed-chip-preview {
  margin: 0;
  color: var(--color-text);
  font-size: .9rem;
  line-height: 1.55;
}
.collapsed-chip-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-top: 2px;
}
.collapsed-chip-hint { color: var(--color-muted); font-size: .72rem; }
.collapsed-chip-action {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-primary-contrast);
  font-size: .74rem;
  font-weight: 720;
  white-space: nowrap;
  transition: background .16s ease, transform .16s ease;
}
.collapsed-chip:hover .collapsed-chip-action { background: var(--color-primary-hover); }
.collapsed-chip-arrow { transition: transform .16s ease; }
.collapsed-chip:hover .collapsed-chip-arrow { transform: translateX(3px); }
.sources--nested { margin-top: 0; }
:deep(.web-results) { margin-top: 6px; }
.collapsible-answer--compact { margin-top: 0; padding-bottom: 20px; }
.collapsible-answer--compact.collapsible-answer--collapsed { margin-top: 0; }
/* The results column is full panel width on wide monitors; cap paragraphs and
   list items at a comfortable reading measure so text doesn't stretch across
   an ultrawide screen, while tables, images and code still span the width. */
@media (min-width: 1100px) {
  :deep(.markdown > p),
  :deep(.markdown li),
  :deep(.markdown > blockquote) {
    max-width: 76ch;
  }
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
