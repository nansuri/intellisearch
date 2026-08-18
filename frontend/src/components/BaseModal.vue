<script setup lang="ts">
import { onBeforeUnmount, watch } from 'vue'

const props = withDefaults(defineProps<{ open: boolean; title: string; busy?: boolean; size?: 'md' | 'lg' }>(), { busy: false, size: 'md' })
const emit = defineEmits<{ (e: 'close'): void }>()

function onKey(e: KeyboardEvent) { if (e.key === 'Escape') emit('close') }
watch(() => props.open, (v) => {
  if (v) window.addEventListener('keydown', onKey)
  else window.removeEventListener('keydown', onKey)
})
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>
<template>
  <div v-if="open" class="modal-backdrop" @mousedown.self="emit('close')">
    <div class="modal" :class="{ 'modal--lg': size === 'lg' }" role="dialog" aria-modal="true" :aria-busy="busy">
      <header class="modal-header">
        <h3>{{ title }}</h3>
        <button class="modal-close" aria-label="Close" @click="emit('close')">&#215;</button>
      </header>
      <div class="modal-body"><slot /></div>
      <footer v-if="$slots.footer" class="modal-footer"><slot name="footer" /></footer>
    </div>
  </div>
</template>