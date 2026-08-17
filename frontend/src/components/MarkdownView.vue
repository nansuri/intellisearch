<script setup lang="ts">
import { computed } from 'vue'
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

const html = computed(() => md.render(props.content || ''))
</script>

<template>
  <div class="markdown" v-html="html" />
</template>
