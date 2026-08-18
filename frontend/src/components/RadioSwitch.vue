<script setup lang="ts">
export interface SwitchOption {
  value: string
  label: string
  title?: string
  hint?: string
}

withDefaults(defineProps<{
  options: SwitchOption[]
  modelValue: string
  size?: 'sm' | 'md'
  variant?: 'segment' | 'pill'
  ariaLabel?: string
}>(), {
  size: 'md',
  variant: 'segment',
  ariaLabel: 'Option',
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<template>
  <div class="radio-switch" :class="[`radio-switch--${size}`, `radio-switch--${variant}`]" role="radiogroup" :aria-label="ariaLabel">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      role="radio"
      :aria-checked="modelValue === option.value"
      :title="option.title"
      :class="{ active: modelValue === option.value }"
      @click="emit('update:modelValue', option.value)"
    >
      <span class="radio-switch-dot" />
      {{ option.label }}
    </button>
  </div>
</template>

<style scoped>
.radio-switch {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: var(--color-surface-subtle);
}
.radio-switch--pill { border-radius: 999px; }
.radio-switch button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 32px;
  padding: 0 12px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: var(--color-muted);
  font-size: .78rem;
  font-weight: 700;
  cursor: pointer;
  transition: color .16s ease, background .16s ease, box-shadow .16s ease;
}
.radio-switch--pill button { border-radius: 999px; }
.radio-switch button:hover:not(.active) { color: var(--color-text); }
.radio-switch button.active {
  color: var(--color-text);
  background: var(--color-surface);
  box-shadow: 0 3px 10px var(--color-shadow);
}
.radio-switch-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--color-muted); transition: background .16s ease; }
.radio-switch button.active .radio-switch-dot { background: var(--color-primary); }
.radio-switch--sm { gap: 4px; padding: 3px; }
.radio-switch--sm button { min-height: 26px; padding: 0 9px; gap: 6px; font-size: .7rem; }
</style>