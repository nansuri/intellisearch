<script setup lang="ts">
import { computed } from 'vue'
import { MINI_APP_SANDBOX, buildDocument, type MiniAppSource } from '../../composables/useMiniAppRunner'

// MiniAppFrame is the single sandboxed execution surface for mini apps: the
// Studio's live preview and the public runner both render through it. Debounced
// srcdoc keeps typing in the IDE cheap while still previewing live.
const props = withDefaults(defineProps<{ source: MiniAppSource; title?: string; debounceMs?: number }>(), {
  title: 'Mini app',
  debounceMs: 300,
})

const srcdoc = computed(() => buildDocument(props.source))
</script>

<template>
  <iframe
    :title="title"
    class="mini-app-frame"
    :sandbox="MINI_APP_SANDBOX"
    :srcdoc="srcdoc"
    allow="camera; microphone"
    frameborder="0"
    referrerpolicy="no-referrer"
  />
</template>

<style scoped>
.mini-app-frame {
  width: 100%;
  height: 100%;
  min-height: 320px;
  border: 0;
  background: #fff;
}
</style>