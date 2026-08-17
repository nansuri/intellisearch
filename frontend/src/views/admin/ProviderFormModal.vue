<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Provider } from '../../services/api'
import BaseModal from '../../components/BaseModal.vue'
import FormField from '../../components/FormField.vue'

const props = defineProps<{ open: boolean; provider: Provider | null }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'save', payload: { name: string; providerType: string; baseUrl: string; model: string; parameters: Record<string, unknown> | null; apiKey?: string; isActive: boolean }): void }>()
const name = ref(''); const providerType = ref<'ollama' | 'openai_compatible'>('ollama')
const baseUrl = ref(''); const model = ref(''); const apiKey = ref(''); const isActive = ref(true)
const parametersText = ref(''); const parametersError = ref('')
const saving = ref(false)

watch(() => props.open, (v) => {
  if (!v) return
  name.value = props.provider?.name || ''
  providerType.value = props.provider?.providerType || 'ollama'
  baseUrl.value = props.provider?.baseUrl || ''
  model.value = props.provider?.model || ''
  apiKey.value = ''
  isActive.value = props.provider?.isActive ?? true
  parametersText.value = props.provider?.parameters ? JSON.stringify(props.provider.parameters, null, 2) : '{\n  "temperature": 0.7\n}'
  parametersError.value = ''
})
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
        <select v-model="providerType" class="text-input">
          <option value="ollama">Ollama</option>
          <option value="openai_compatible">OpenAI-compatible</option>
        </select>
      </FormField>
      <FormField label="Base URL" hint="e.g. http://localhost:11434/v1 for Ollama" :error="baseUrl ? '' : undefined">
        <input v-model="baseUrl" class="text-input" type="url" required placeholder="http://localhost:11434/v1" />
      </FormField>
      <FormField label="Model" :error="model ? '' : undefined">
        <input v-model="model" class="text-input" required placeholder="llama3.2" />
      </FormField>
      <FormField v-if="providerType === 'openai_compatible'" label="API key" hint="Stored encrypted; leave blank to keep the existing key">
        <input v-model="apiKey" class="text-input" type="password" :placeholder="provider ? 'Keep current key' : 'sk-…'" />
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