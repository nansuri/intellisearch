<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Provider, ProviderType } from '../../services/api'
import BaseModal from '../../components/BaseModal.vue'
import FormField from '../../components/FormField.vue'

const props = defineProps<{ open: boolean; provider: Provider | null }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'save', payload: { name: string; providerType: string; baseUrl: string; model: string; parameters: Record<string, unknown> | null; apiKey?: string; isActive: boolean }): void }>()
const name = ref(''); const providerType = ref<ProviderType>('ollama')
const baseUrl = ref(''); const model = ref(''); const apiKey = ref(''); const isActive = ref(true)
const parametersText = ref(''); const parametersError = ref('')
const saving = ref(false)
let lastType: ProviderType = 'ollama'

const TYPE_PRESETS: Record<ProviderType, { label: string; baseUrl: string; model: string; needsKey: boolean; baseUrlHint: string }> = {
  ollama: { label: 'Ollama', baseUrl: 'http://localhost:11434', model: 'llama3.2', needsKey: false, baseUrlHint: 'Local server — no API key needed' },
  openai_compatible: { label: 'OpenAI-compatible', baseUrl: 'https://api.openai.com/v1', model: 'gpt-4o-mini', needsKey: true, baseUrlHint: 'Any OpenAI-compatible /v1/chat/completions endpoint' },
  pollinations: { label: 'Pollinations.ai', baseUrl: 'https://text.pollinations.ai', model: 'openai', needsKey: false, baseUrlHint: 'Keyless — no API key needed' },
  huggingface: { label: 'Hugging Face', baseUrl: 'https://router.huggingface.co/v1', model: 'Qwen/Qwen3-70B-Instruct', needsKey: true, baseUrlHint: 'Use an hf_… token from Hugging Face' },
}

const selectedPreset = computed(() => TYPE_PRESETS[providerType.value])
const modelPlaceholder = computed(() => selectedPreset.value.model)

watch(() => props.open, (v) => {
  if (!v) return
  name.value = props.provider?.name || ''
  providerType.value = (props.provider?.providerType as ProviderType) || 'ollama'
  lastType = providerType.value
  baseUrl.value = props.provider?.baseUrl || ''
  model.value = props.provider?.model || ''
  apiKey.value = ''
  isActive.value = props.provider?.isActive ?? true
  parametersText.value = props.provider?.parameters ? JSON.stringify(props.provider.parameters, null, 2) : '{\n  "temperature": 0.7\n}'
  parametersError.value = ''
})

// When the user picks a new provider type, prefill the endpoint + model unless
// they already typed custom values (or the field still holds the previous
// type's preset).
function onTypeChange() {
  const preset = TYPE_PRESETS[providerType.value]
  const previous = TYPE_PRESETS[lastType]
  if (!baseUrl.value || (previous && baseUrl.value === previous.baseUrl)) baseUrl.value = preset.baseUrl
  if (!model.value || (previous && model.value === previous.model)) model.value = preset.model
  lastType = providerType.value
}

function parseParameters(): Record<string, unknown> | null | undefined {
  const t = parametersText.value.trim()
  if (!t) return null
  try {
    const parsed = JSON.parse(t)
    if (typeof parsed !== 'object' || parsed === null) throw new Error()
    return parsed as Record<string, unknown>
  } catch { parametersError.value = 'Parameters must be valid JSON object.'; return undefined }
}
function submit() {
  const parameters = parseParameters()
  if (parameters === undefined) return
  saving.value = true
  emit('save', { name: name.value.trim(), providerType: providerType.value, baseUrl: baseUrl.value.trim(), model: model.value.trim(), parameters, isActive: isActive.value, ...(apiKey.value ? { apiKey: apiKey.value } : {}) })
  setTimeout(() => (saving.value = false), 400)
}
</script>
<template>
  <BaseModal :open="open" :title="provider ? 'Edit provider' : 'New provider'" @close="emit('close')">
    <form class="admin-form" @submit.prevent="submit">
      <FormField label="Name" :error="name ? '' : undefined">
        <input v-model="name" class="text-input" required placeholder="Local Ollama" />
      </FormField>
      <FormField label="Provider type">
        <select v-model="providerType" class="text-input" @change="onTypeChange">
          <option v-for="(preset, type) in TYPE_PRESETS" :key="type" :value="type">{{ preset.label }}</option>
        </select>
      </FormField>
      <FormField label="Base URL" :hint="selectedPreset.baseUrlHint" :error="baseUrl ? '' : undefined">
        <input v-model="baseUrl" class="text-input" type="url" required :placeholder="selectedPreset.baseUrl" />
      </FormField>
      <FormField label="Model" :error="model ? '' : undefined">
        <input v-model="model" class="text-input" required :placeholder="modelPlaceholder" />
      </FormField>
      <FormField v-if="selectedPreset.needsKey" label="API key" hint="Stored encrypted; leave blank to keep the existing key">
        <input v-model="apiKey" class="text-input" type="password" :placeholder="provider ? 'Keep current key' : 'sk-… / hf_…'" />
      </FormField>
      <FormField label="Model parameters (JSON)" :error="parametersError">
        <textarea v-model="parametersText" class="text-input text-area" rows="5" spellcheck="false" />
      </FormField>
      <label class="checkbox-row"><input v-model="isActive" type="checkbox" /><span>Active provider (routed to next request)</span></label>
      <div class="modal-submit-row">
        <button type="button" class="base-button button-secondary" @click="emit('close')">Cancel</button>
        <button type="submit" class="base-button button-primary" :disabled="saving || !name || !baseUrl || !model">{{ saving ? 'Saving…' : provider ? 'Save changes' : 'Create provider' }}</button>
      </div>
    </form>
  </BaseModal>
</template>
