<script setup lang="ts">
import { ref, watch } from 'vue'

// MiniAppEditor is the reusable labeled code field of the Mini App Studio's IDE
// (one per HTML/CSS/JS). A per-field debounce keeps <iframe srcdoc> updates from
// re-creating the frame on every keystroke.
const props = withDefaults(defineProps<{ label: string; modelValue: string; language?: string; rows?: number; placeholder?: string }>(), {
  language: 'text',
  rows: 8,
  placeholder: '',
})
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const value = ref(props.modelValue)
watch(
  () => props.modelValue,
  (next) => { value.value = next },
)
watch(value, (next) => {
  emit('update:modelValue', next)
})
</script>

<template>
  <label class="mini-app-editor">
    <span class="mini-app-editor-label">
      {{ label }}
      <code v-if="language && language !== 'text'" class="mini-app-editor-lang">{{ language }}</code>
    </span>
    <textarea v-model="value" :rows="rows" :placeholder="placeholder" class="mini-app-editor-area" :spellcheck="false" />
  </label>
</template>

<style scoped>
.mini-app-editor {
  display: grid;
  gap: 6px;
  min-width: 0;
}
.mini-app-editor-label {
  display: flex;
  align-items: baseline;
  gap: 8px;
  color: var(--color-muted);
  font-size: .7rem;
  font-weight: 720;
  letter-spacing: .06em;
  text-transform: uppercase;
}
.mini-app-editor-lang {
  color: var(--color-primary);
  font-size: .68rem;
  letter-spacing: .02em;
  text-transform: none;
}
.mini-app-editor-area {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  background: var(--color-code-bg, var(--color-surface-subtle));
  color: var(--color-text);
  font: 12.5px/1.55 var(--font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  resize: vertical;
}
.mini-app-editor-area:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-primary) 22%, transparent);
}
</style>