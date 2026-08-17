<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import MarkdownIt from 'markdown-it'
import * as emoji from 'markdown-it-emoji'

const props = defineProps<{ content: string }>()

// markdown-it-emoji v3 exposes bare/light/full as named exports (no default
// or `emoji` export) — the namespace's `full` plugin is the complete emoji set.
const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  typographer: true,
}).use(emoji.full)

const defaultLinkOpen = md.renderer.rules.link_open
md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  token.attrSet('target', '_blank')
  token.attrSet('rel', 'noopener noreferrer')
  return defaultLinkOpen
    ? defaultLinkOpen(tokens, idx, options, env, self)
    : self.renderToken(tokens, idx, options)
}

const defaultImage = md.renderer.rules.image
md.renderer.rules.image = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  token.attrSet('loading', 'lazy')
  token.attrSet('referrerpolicy', 'no-referrer')
  return defaultImage
    ? defaultImage(tokens, idx, options, env, self)
    : self.renderToken(tokens, idx, options)
}

const root = ref<HTMLElement | null>(null)
const html = computed(() => md.render(props.content || ''))

// --- Mermaid rendering -------------------------------------------------------
// AI answers may include ```mermaid fences (flowcharts, UML, timelines, ...).
// The mermaid library is heavy, so it is lazy-imported only when such a fence
// is present; failures fall back to showing the raw code block.
let mermaidApi: any = null
let renderSeq = 0
let observer: MutationObserver | null = null
// Rendered wrappers keyed by their mermaid source, so a theme flip can re-render.
const diagrams = new Map<HTMLElement, string>()

async function ensureMermaid() {
  if (mermaidApi) return true
  try {
    mermaidApi = (await import('mermaid')).default
    return true
  } catch {
    return false
  }
}

function initializeMermaid() {
  const dark = document.documentElement.dataset.theme === 'dark'
  mermaidApi.initialize({
    startOnLoad: false,
    theme: dark ? 'dark' : 'default',
    securityLevel: 'loose',
    fontFamily: 'inherit',
  })
}

async function renderOne(wrap: HTMLElement, source: string) {
  const id = `mmd-${Date.now()}-${++renderSeq}`
  try {
    const { svg } = await mermaidApi.render(id, source)
    wrap.innerHTML = svg
  } catch {
    diagrams.delete(wrap)
    const fallback = document.createElement('pre')
    fallback.className = 'mermaid-fallback'
    fallback.textContent = source
    wrap.replaceWith(fallback)
  }
}

async function renderMermaidBlocks() {
  const host = root.value
  if (!host) return
  // Drop wrappers from a previous content pass that are no longer mounted.
  for (const wrap of Array.from(diagrams.keys())) {
    if (!wrap.isConnected) diagrams.delete(wrap)
  }
  const blocks = Array.from(host.querySelectorAll<HTMLElement>('pre code.language-mermaid'))
  if (!blocks.length) return
  if (!(await ensureMermaid())) return
  initializeMermaid()
  for (const block of blocks) {
    const pre = block.parentElement
    if (!pre) continue
    const source = block.textContent || ''
    if (!source.trim()) continue
    const wrap = document.createElement('div')
    wrap.className = 'mermaid-diagram'
    pre.replaceWith(wrap)
    diagrams.set(wrap, source)
    await renderOne(wrap, source)
  }
}

async function reRenderAll() {
  if (!diagrams.size || !(await ensureMermaid())) return
  initializeMermaid()
  for (const [wrap, source] of Array.from(diagrams)) {
    await renderOne(wrap, source)
  }
}

watch(html, async () => {
  await nextTick()
  void renderMermaidBlocks()
})

// Re-render diagrams when the theme flips (mermaid theme is fixed at render).
function observeTheme() {
  observer = new MutationObserver(() => {
    void reRenderAll()
  })
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
}
onBeforeUnmount(() => observer?.disconnect())
watch(
  () => root.value,
  (el) => {
    if (el && !observer) observeTheme()
  },
  { immediate: true },
)
</script>

<template>
  <div ref="root" class="markdown" v-html="html" />
</template>

<style scoped>
.mermaid-diagram {
  display: flex;
  justify-content: center;
  margin: 16px 0;
  padding: 14px;
  overflow-x: auto;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: var(--color-surface-subtle);
}
.mermaid-diagram :deep(svg) {
  max-width: 100%;
  height: auto;
}
</style>
