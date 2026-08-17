<script setup lang="ts">
type Source = { position: number; title: string; url: string; domain: string; snippet: string }
const props = defineProps<{ source: Source }>()
</script>
<template>
  <article class="source-card">
    <span class="source-number">{{ props.source.position }}</span>
    <div class="source-copy"><small>{{ props.source.domain }}</small><a :href="props.source.url" target="_blank" rel="noreferrer">{{ props.source.title }}</a><p>{{ props.source.snippet }}</p><span class="source-url">{{ props.source.url }}</span></div>
  </article>
</template>
