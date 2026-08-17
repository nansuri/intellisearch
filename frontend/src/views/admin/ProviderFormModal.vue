<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { listOllamaModels, getOllamaHealth, type OllamaHealth, type OllamaModel, type Provider, type ProviderType } from '../../services/api'
import BaseModal from '../../components/BaseModal.vue'
import FormField from '../../components/FormField.vue'

const props = defineProps<{ open: boolean; provider: Provider | null }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'save', payload: { name: string; providerType: string; baseUrl: string; model: string; parameters: Record<string, unknown> | null; apiKey?: string; isActive: boolean }): void }>()
const name = ref(''); const providerType = ref<ProviderType>('ollama')
const baseUrl = ref(''); const model = ref(''); const apiKey = ref(''); const isActive = ref(true)
const parametersText = ref(''); const parametersError = ref('')
const saving = ref(false)
let lastType: ProviderType = 'ollama'

// Ollama server introspection (proxied through the Go API — the browser never
// talks to Ollama directly).
const ollamaModels = ref<OllamaModel[]>([])
const ollamaHealth = ref<OllamaHealth | null>(null)
const ollamaStatus = ref<'idle' | 'loading' | 'ok' | 'error'>('idle')
const ollamaError = ref('')

const TYPE_PRESETS: Record<ProviderType, { label: string; baseUrl: string; model: string; needsKey: boolean; baseUrlHint: string }> = {
  ollama: { label: 'Ollama', baseUrl: 'http://localhost:11434', model: 'llama3.2', needsKey: false, baseUrlHint: 'Local server — no API key needed' },
  openai_compatible: { label: 'OpenAI-compatible', baseUrl: 'https://api.openai.com/v1', model: 'gpt-4o-mini', needsKey: true, baseUrlHint: 'Any OpenAI-compatible /v1/chat/completions endpoint' },
  pollinations: { label: 'Pollinations.ai', baseUrl: 'https://gen.pollinations.ai', model: 'openai', needsKey: true, baseUrlHint: 'API key required — get one at enter.pollinations.ai' },
  huggingface: { label: 'Hugging Face', baseUrl: 'https://router.huggingface.co/v1', model: 'Qwen/Qwen3-70B-Instruct', needsKey: true, baseUrlHint: 'Use an hf_… token from Hugging Face' },
}

const selectedPreset = computed(() => TYPE_PRESETS[providerType.value])
const modelPlaceholder = computed(() => selectedPreset.value.model)
const isOllama = computed(() => providerType.value === 'ollama')
const ollamaConnected = computed(() => ollamaStatus.value === 'ok')

function resetOllama() {
  ollamaModels.value = []
  ollamaHealth.value = null
  ollamaStatus.value = 'idle'
  ollamaError.value = ''
}

async function loadOllama() {
  const target = baseUrl.value.trim()
  if (!target) return
  ollamaStatus.value = 'loading'
  ollamaError.value = ''
  try {
    const [models, health] = await Promise.all([listOllamaModels(target), getOllamaHealth(target)])
    ollamaModels.value = models.models
    ollamaHealth.value = health
    ollamaStatus.value = 'ok'
    // Auto-pick the first model only when nothing is typed yet.
    if (!model.value && models.models.length) model.value = models.models[0].name
  } catch (e) {
    ollamaStatus.value = 'error'
    ollamaError.value = (e as Error).message
  }
}

function onBaseUrlChange() {
  if (!isOllama.value) return
  resetOllama()
  if (baseUrl.value.trim()) loadOllama()
}

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
  resetOllama()
  if (isOllama.value && baseUrl.value.trim()) loadOllama()
})

// When the user picks a new provider type, prefill the endpoint + model unless
// they already typed custom values (or the field still holds the previous
// type's preset). Ollama gains a model picker, other types drop it.
function onTypeChange() {
  const preset = TYPE_PRESETS[providerType.value]
  const previous = TYPE_PRESETS[lastType]
  if (!baseUrl.value || (previous && baseUrl.value === previous.baseUrl)) baseUrl.value = preset.baseUrl
  if (!model.value || (previous && model.value === previous.model)) model.value = preset.model
  lastType = providerType.value
  if (isOllama.value) {
    resetOllama()
    if (baseUrl.value.trim()) loadOllama()
  } else {
    resetOllama()
  }
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
        <input v-model="baseUrl" class="text-input" type="url" required :placeholder="selectedPreset.baseUrl" @change="onBaseUrlChange" />
      </FormField>
      <FormField label="Model" :hint="isOllama && ollamaConnected ? `${ollamaModels.length} models available — pick one or type a name` : undefined" :error="model ? '' : undefined">
        <input v-model="model" class="text-input" required :placeholder="modelPlaceholder" :list="isOllama && ollamaModels.length ? 'ollama-model-list' : undefined" />
        <datalist v-if="isOllama && ollamaModels.length" id="ollama-model-list">
          <option v-for="m in ollamaModels" :key="m.name" :value="m.name" />
        </datalist>
      </FormField>
      <FormField v-if="isOllama" label="Ollama server" hint="Models and health are fetched server-side through the API.">
        <button type="button" class="base-button button-secondary ollama-load" :disabled="!baseUrl.trim() || ollamaStatus === 'loading'" @click="loadOllama">
          {{ ollamaStatus === 'loading' ? 'Connecting…' : ollamaConnected ? 'Refresh' : 'Load models & health' }}
        </button>
      </FormField>
      <div v-if="isOllama && ollamaConnected" class="ollama-status">
        <div class="ollama-status-row">
          <span class="ollama-dot" />
          <span>Connected · Ollama {{ ollamaHealth?.version || 'unknown' }} · {{ ollamaModels.length }} model{{ ollamaModels.length === 1 ? '' : 's' }} available</span>
        </div>
        <template v-if="ollamaHealth?.runningModels?.length">
          <div class="ollama-running-title">Loaded in memory</div>
          <div v-for="m in ollamaHealth.runningModels" :key="m.name" class="ollama-running">
            <span class="ollama-running-name">{{ m.name }}</span>
            <span v-if="m.cpu" class="ollama-running-stat">cpu {{ m.cpu }}</span>
            <span v-if="m.gpu" class="ollama-running-stat">gpu {{ m.gpu }}</span>
            <span v-if="m.memory" class="ollama-running-stat">{{ m.memory }}</span>
          </div>
        </template>
        <p v-else class="ollama-note">No models loaded in memory right now.</p>
      </div>
      <p v-else-if="isOllama && ollamaStatus === 'error'" class="ollama-error" role="alert">{{ ollamaError }}</p>
      <FormField v-if="selectedPreset.needsKey" label="API key" hint="Stored encrypted; leave blank to keep the existing key">
        <input v-model="apiKey" class="text-input" type="password" :placeholder="provider ? 'Keep current key' : (providerType === 'huggingface' ? 'hf_…' : 'pk_… / sk_…')" />
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

<style scoped>
.ollama-load { min-height: 38px; }
.ollama-status { display: grid; gap: 8px; margin-top: -6px; padding: 12px 14px; border: 1px solid color-mix(in srgb, #34d399 35%, var(--color-border)); border-radius: 12px; background: color-mix(in srgb, #34d399 7%, var(--color-surface)); font-size: .8rem; }
.ollama-status-row { display: flex; align-items: center; gap: 8px; font-weight: 650; }
.ollama-dot { width: 8px; height: 8px; flex: 0 0 auto; border-radius: 50%; background: #34d399; box-shadow: 0 0 0 3px color-mix(in srgb, #34d399 22%, transparent); }
.ollama-running-title { margin-top: 4px; color: var(--color-muted); font-size: .68rem; font-weight: 720; letter-spacing: .07em; text-transform: uppercase; }
.ollama-running { display: flex; flex-wrap: wrap; align-items: baseline; gap: 4px 12px; }
.ollama-running-name { font-weight: 680; }
.ollama-running-stat { color: var(--color-muted); font-size: .74rem; font-variant-numeric: tabular-nums; }
.ollama-note { margin: 0; color: var(--color-muted); font-size: .76rem; }
.ollama-error { margin: -6px 0 0; padding: 10px 12px; border: 1px solid color-mix(in srgb, var(--color-danger) 35%, var(--color-border)); border-radius: 10px; background: color-mix(in srgb, var(--color-danger) 8%, var(--color-surface)); color: var(--color-danger); font-size: .78rem; line-height: 1.5; }
</style>
