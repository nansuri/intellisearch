<script setup lang="ts">
import { computed } from 'vue'
import MarkdownView from './MarkdownView.vue'
import SourceCard from './SourceCard.vue'
import { answerPreview } from '../utils/answerSummary'
import type { Source } from '../services/api'

const props = withDefaults(defineProps<{
  label: string
  answer: string
  sources?: Source[]
  collapsed?: boolean
  highlighted?: boolean
  showSources?: boolean
  compact?: boolean
}>(), {
  sources: () => [],
  collapsed: false,
  highlighted: false,
  showSources: true,
  compact: false,
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
      <span class="collapsed-chip-label">{{ label }}</span>
      <p class="collapsed-chip-preview">{{ preview }}</p>
      <span class="collapsed-chip-meta">
        <span v-if="sources.length">{{ sources.length }} source{{ sources.length === 1 ? '' : 's' }}</span>
        <span class="collapsed-chip-action">Show full answer</span>
      </span>
    </button>

    <template v-else>
      <div class="collapsible-answer-head">
        <div class="section-label">{{ label }}</div>
        <button v-if="canCollapse" type="button" class="collapse-toggle" @click="collapse">Collapse</button>
      </div>
      <MarkdownView v-if="answer" :content="answer" />
      <p v-else>No answer yet. Try asking again.</p>

      <section v-if="showSources && sources.length" class="sources sources--nested">
        <div class="sources-heading"><h2>Sources</h2><span>{{ sources.length }} results</span></div>
        <SourceCard v-for="source in sources" :key="source.position" :source="source" />
      </section>
    </template>
  </article>
</template>

<style scoped>
.collapsible-answer {
  display: grid;
  gap: 16px;
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
  display: grid;
  gap: 6px;
  width: 100%;
  padding: 14px 16px;
  border: 1px solid var(--color-border);
  border-radius: 14px;
  background: var(--color-surface-subtle);
  color: inherit;
  cursor: pointer;
  text-align: left;
  transition: border-color .16s ease, background .16s ease, transform .16s ease;
}
.collapsed-chip:hover {
  border-color: color-mix(in srgb, var(--color-primary) 40%, var(--color-border));
  background: color-mix(in srgb, var(--color-primary) 5%, var(--color-surface-subtle));
  transform: translateY(-1px);
}
.collapsed-chip-label {
  color: var(--color-primary);
  font-size: .68rem;
  font-weight: 760;
  letter-spacing: .08em;
  text-transform: uppercase;
}
.collapsed-chip-preview {
  margin: 0;
  color: var(--color-text);
  font-size: .88rem;
  line-height: 1.5;
}
.collapsed-chip-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  color: var(--color-muted);
  font-size: .72rem;
}
.collapsed-chip-action { color: var(--color-primary); font-weight: 650; }
.sources--nested { margin-top: 8px; }
.collapsible-answer--compact { margin-top: 0; padding-bottom: 20px; }
.collapsible-answer--compact.collapsible-answer--collapsed { margin-top: 0; }
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
