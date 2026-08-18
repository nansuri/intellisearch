<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Source } from '../services/api'
import { highlightTerms } from '../utils/highlight'
import { clampSnippet } from '../utils/snippet'
import { SNIPPET_MAX_CHARS } from '../config/app'

const props = withDefaults(defineProps<{ source: Source; query?: string; maxChars?: number }>(), {
  query: '',
  maxChars: SNIPPET_MAX_CHARS,
})

const expanded = ref(false)
watch(() => props.source.snippet, () => { expanded.value = false })

const truncated = computed(() => clampSnippet(props.source.snippet || '', props.maxChars))
const isTruncated = computed(() => truncated.value !== (props.source.snippet || '').trim())

const snippet = computed(() => {
  const text = expanded.value ? props.source.snippet || '' : truncated.value
  return highlightTerms(text, props.query || '')
})
const iconLetter = computed(() => (props.source.domain || 'w').trim().charAt(0).toUpperCase())
</script>

<template>
  <article class="web-result">
    <a class="web-result-link" :href="props.source.url" target="_blank" rel="noreferrer">
      <span class="web-result-title-row">
        <span class="web-result-icon" aria-hidden="true">{{ iconLetter }}</span>
        <span class="web-result-title">{{ props.source.title }}</span>
      </span>
      <span class="web-result-url">{{ props.source.domain }}</span>
      <p class="web-result-snippet" v-html="snippet"></p>
    </a>
    <button
      v-if="isTruncated"
      type="button"
      class="web-result-more"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      {{ expanded ? 'Show less' : 'Read more' }}
    </button>
  </article>
</template>

<style scoped>
.web-result { padding: 16px 0; border-bottom: 1px solid var(--color-border); }
.web-result:last-of-type { border-bottom: 0; }
.web-result-link { display: block; text-decoration: none; }
.web-result-title-row { display: flex; align-items: center; gap: 10px; min-width: 0; }
.web-result-icon {
  display: inline-grid;
  place-items: center;
  width: 26px;
  height: 26px;
  flex: 0 0 auto;
  border-radius: 8px;
  background: linear-gradient(135deg, var(--color-surface-subtle), color-mix(in srgb, var(--color-primary) 18%, var(--color-surface-subtle)));
  border: 1px solid var(--color-border);
  color: var(--color-muted);
  font-size: .72rem;
  font-weight: 780;
  text-transform: uppercase;
}
.web-result-title {
  min-width: 0;
  overflow: hidden;
  color: var(--color-primary);
  font-size: 1rem;
  font-weight: 660;
  letter-spacing: -.015em;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.web-result-link:hover .web-result-title { text-decoration: underline; }
.web-result-url {
  display: block;
  margin-top: 3px;
  color: #188038;
  font-size: .74rem;
  font-weight: 520;
}
[data-theme="dark"] .web-result-url { color: #7ee08b; }
.web-result-snippet {
  margin: 4px 0 0;
  overflow-wrap: anywhere;
  color: var(--color-muted);
  font-size: .86rem;
  line-height: 1.55;
}
.web-result-snippet :deep(mark) {
  padding: 0 1px;
  border-radius: 3px;
  background: color-mix(in srgb, var(--color-primary) 22%, transparent);
  color: var(--color-text);
  font-weight: 650;
}
.web-result-more {
  margin: 6px 0 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--color-primary);
  font-size: .78rem;
  font-weight: 680;
  cursor: pointer;
}
.web-result-more:hover { text-decoration: underline; }
</style>