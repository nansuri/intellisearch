<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { getPollinationsAccount, getPollinationsDailyUsage, getPollinationsModels, listOllamaModels, getOllamaHealth, type OllamaHealth, type OllamaModel, type PollinationsAccount, type PollinationsDailyUsage as PollDaily, type PollinationsModel, type Provider, type ProviderType } from '../../services/api'
import BaseModal from '../../components/BaseModal.vue'
import FormField from '../../components/FormField.vue'

const props = defineProps<{ open: boolean; provider: Provider | null }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'save', payload: { name: string; providerType: string; baseUrl: string; model: string; parameters: Record<string, unknown> | null; apiKey?: string; isActive: boolean }): void }>()
const name = ref(''); const providerType = ref<ProviderType>('ollama')
const baseUrl = ref(''); const model = ref(''); const apiKey = ref(''); const isActive = ref(true)
const parametersText = ref(''); const parametersError = ref('')
const saving = ref(false)
// When true the model field is a plain input; otherwise (Ollama models loaded)
// it is a real <select> dropdown. Selecting "Type a custom model…" flips this.
const modelCustom = ref(false)
let lastType: ProviderType = 'ollama'

// Ollama server introspection (proxied through the Go API — the browser never
// talks to Ollama directly).
const ollamaModels = ref<OllamaModel[]>([])
const ollamaHealth = ref<OllamaHealth | null>(null)
const ollamaStatus = ref<'idle' | 'loading' | 'ok' | 'error'>('idle')
const ollamaError = ref('')

// Pollinations account introspection (also proxied through the Go API — the
// browser never calls Pollinations directly). The provider's stored API key is
// decrypted server-side; for an unsaved provider the typed key is used.
const pollAccount = ref<PollinationsAccount | null>(null)
const pollDaily = ref<PollDaily[]>([])
const pollModels = ref<PollinationsModel[]>([])
const pollStatus = ref<'idle' | 'loading' | 'ok' | 'error'>('idle')
const pollError = ref('')

const isPollinations = computed(() => providerType.value === 'pollinations')
// The provider being edited is a saved Pollinations provider → resolve its key
// server-side by id. Otherwise fall back to the typed key + base URL.
const pollCredentials = computed(() => {
  if (props.provider?.providerType === 'pollinations' && props.provider.id) return { providerId: props.provider.id }
  return { apiKey: apiKey.value.trim(), baseUrl: baseUrl.value.trim() || 'https://gen.pollinations.ai' }
})
const pollCanLoad = computed(() => Boolean((pollCredentials.value as { providerId?: string }).providerId) || Boolean((pollCredentials.value as { apiKey?: string }).apiKey))
const pollConnected = computed(() => pollStatus.value === 'ok' && pollAccount.value !== null)
const pollUsageSummary = computed(() => {
  const totalRequests = pollDaily.value.reduce((sum, row) => sum + row.requests, 0)
  const totalCost = pollDaily.value.reduce((sum, row) => sum + row.cost_usd, 0)
  return { totalRequests, totalCost }
})

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

// Real dropdown once the server's models are known (Ollama /api/tags or
// Pollinations /v1/models); otherwise the field stays a text input.
const showModelSelect = computed(() =>
  (isOllama.value && ollamaConnected.value && ollamaModels.value.length > 0) ||
  (isPollinations.value && pollConnected.value && pollModels.value.length > 0),
)
const modelSelectCount = computed(() => (isOllama.value ? ollamaModels.value.length : pollModels.value.length))
// A provider being edited may reference a model that no longer exists on the
// server — keep it selectable so the form never shows a blank dropdown.
const modelOptions = computed(() => {
  const names = isOllama.value ? ollamaModels.value.map((m) => m.name) : pollModels.value.map((m) => m.id)
  if (model.value && !names.includes(model.value)) return [model.value, ...names]
  return names
})

const CUSTOM_MODEL = '__custom_model__'
function onModelSelectChange() {
  if (model.value === CUSTOM_MODEL) {
    modelCustom.value = true
    model.value = ''
  }
}

function resetOllama() {
  ollamaModels.value = []
  ollamaHealth.value = null
  ollamaStatus.value = 'idle'
  ollamaError.value = ''
  modelCustom.value = false
}

function resetPollinations() {
  pollAccount.value = null
  pollDaily.value = []
  pollModels.value = []
  pollStatus.value = 'idle'
  pollError.value = ''
}

async function loadPollinations() {
  if (!pollCanLoad.value) return
  pollStatus.value = 'loading'
  pollError.value = ''
  const input = pollCredentials.value
  try {
    const [account, daily, models] = await Promise.all([
      getPollinationsAccount(input),
      getPollinationsDailyUsage(input, 14),
      getPollinationsModels(input),
    ])
    pollAccount.value = account
    pollDaily.value = daily.usage || []
    pollModels.value = models.models || []
    pollStatus.value = 'ok'
    // Auto-pick the first model only when nothing is typed yet.
    if (!model.value && pollModels.value.length) model.value = pollModels.value[0].id
  } catch (e) {
    pollStatus.value = 'error'
    pollError.value = (e as Error).message
  }
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
  if (isOllama.value) {
    resetOllama()
    if (baseUrl.value.trim()) loadOllama()
    return
  }
  if (isPollinations.value) {
    resetPollinations()
    loadPollinations()
  }
}

function onApiKeyChange() {
  if (isPollinations.value) {
    resetPollinations()
    if (apiKey.value.trim()) loadPollinations()
  }
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
  modelCustom.value = false
  resetOllama()
  resetPollinations()
  if (isOllama.value && baseUrl.value.trim()) loadOllama()
  if (isPollinations.value) loadPollinations()
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
  resetOllama()
  resetPollinations()
  if (isOllama.value) {
    if (baseUrl.value.trim()) loadOllama()
  } else if (isPollinations.value) {
    loadPollinations()
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
  <BaseModal :open="open" :title="provider ? 'Edit provider' : 'New provider'" size="lg" @close="emit('close')">
    <form class="admin-form provider-form" @submit.prevent="submit">
      <div class="provider-section">
        <div class="setting-title"><div><div class="section-label">Connection</div><h2>Endpoint</h2><p>Where requests are sent and which model answers.</p></div></div>
        <div class="form-grid-2">
          <FormField label="Name" :error="name ? '' : undefined">
            <input v-model="name" class="text-input" required placeholder="Local Ollama" />
          </FormField>
          <FormField label="Provider type">
            <select v-model="providerType" class="text-input" @change="onTypeChange">
              <option v-for="(preset, type) in TYPE_PRESETS" :key="type" :value="type">{{ preset.label }}</option>
            </select>
          </FormField>
        </div>
        <FormField label="Base URL" :hint="selectedPreset.baseUrlHint" :error="baseUrl ? '' : undefined">
          <input v-model="baseUrl" class="text-input" type="url" required :placeholder="selectedPreset.baseUrl" @change="onBaseUrlChange" />
        </FormField>
        <FormField label="Model" :hint="showModelSelect && !modelCustom ? `${modelSelectCount} models available` : undefined" :error="model ? '' : undefined">
          <select v-if="showModelSelect && !modelCustom" v-model="model" class="text-input" required @change="onModelSelectChange">
            <option v-for="m in modelOptions" :key="m" :value="m">{{ m }}</option>
            <option :value="CUSTOM_MODEL">Type a custom model…</option>
          </select>
          <input v-else v-model="model" class="text-input" required :placeholder="modelPlaceholder" />
        </FormField>
        <FormField v-if="selectedPreset.needsKey" label="API key" hint="Stored encrypted; leave blank to keep the existing key">
          <input v-model="apiKey" class="text-input" type="password" :placeholder="provider ? 'Keep current key' : (providerType === 'huggingface' ? 'hf_…' : 'pk_… / sk_…')" @change="onApiKeyChange" />
        </FormField>
      </div>

      <div v-if="isOllama || isPollinations" class="provider-section">
        <div class="setting-title"><div><div class="section-label">Server integration</div><h2>{{ isOllama ? 'Ollama server' : 'Pollinations account' }}</h2><p>Live introspection is proxied through the API — the browser never calls the provider directly.</p></div></div>
        <div v-if="isOllama">
          <button type="button" class="base-button button-secondary ollama-load" :disabled="!baseUrl.trim() || ollamaStatus === 'loading'" @click="loadOllama">
            {{ ollamaStatus === 'loading' ? 'Connecting…' : ollamaConnected ? 'Refresh' : 'Load models & health' }}
          </button>
          <div v-if="ollamaConnected" class="ollama-status">
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
          <p v-else-if="ollamaStatus === 'error'" class="ollama-error" role="alert">{{ ollamaError }}</p>
        </div>
        <div v-else-if="isPollinations">
          <button type="button" class="base-button button-secondary ollama-load" :disabled="!pollCanLoad || pollStatus === 'loading'" @click="loadPollinations">
            {{ pollStatus === 'loading' ? 'Checking…' : pollConnected ? 'Refresh account & usage' : 'Load credits & usage' }}
          </button>
          <div v-if="pollConnected" class="ollama-status poll-status">
            <div class="ollama-status-row">
              <span class="ollama-dot" />
              <span>Balance <strong>{{ pollAccount?.balance?.toLocaleString(undefined, { maximumFractionDigits: 2 }) }}</strong> pollen</span>
            </div>
            <template v-if="pollAccount?.key">
              <div class="poll-key-row">
                <span class="poll-key-type">{{ pollAccount.key.type }} key</span>
                <span v-if="pollAccount.key.expiresAt" class="poll-key-meta">expires {{ new Date(pollAccount.key.expiresAt).toLocaleDateString() }}</span>
                <span v-else class="poll-key-meta">never expires</span>
                <span v-if="pollAccount.key.pollenBudget != null" class="poll-key-meta">key budget {{ pollAccount.key.pollenBudget }}</span>
              </div>
              <p v-if="pollAccount.key.permissions?.account?.length" class="ollama-note">account scopes: {{ pollAccount.key.permissions.account.join(', ') }}</p>
              <p v-else-if="!pollAccount.key.valid" class="poll-key-invalid">This API key is not valid — check it on enter.pollinations.ai.</p>
            </template>
            <template v-if="pollAccount?.profile">
              <div class="ollama-running-title">Account</div>
              <div class="ollama-running">
                <span class="ollama-running-name">{{ pollAccount.profile.githubUsername || 'Pollinations user' }}</span>
                <span v-if="pollAccount.profile.name" class="ollama-running-stat">{{ pollAccount.profile.name }}</span>
                <span v-if="pollAccount.profile.email" class="ollama-running-stat">{{ pollAccount.profile.email }}</span>
              </div>
            </template>
            <template v-if="pollDaily.length">
              <div class="ollama-running-title">Usage · last {{ pollDaily.length }} day{{ pollDaily.length === 1 ? '' : 's' }}</div>
              <div class="ollama-running">
                <span class="ollama-running-name">{{ pollUsageSummary.totalRequests }} requests</span>
                <span class="ollama-running-stat">≈ ${{ pollUsageSummary.totalCost.toFixed(4) }}</span>
              </div>
              <div class="poll-daily-table">
                <div v-for="row in pollDaily.slice(0, 7)" :key="row.date" class="poll-daily-row">
                  <span class="poll-daily-date">{{ row.date }}</span>
                  <span class="poll-daily-model">{{ row.model || '—' }}</span>
                  <span class="poll-daily-stat">{{ row.requests }} req</span>
                  <span class="poll-daily-stat">${{ row.cost_usd.toFixed(4) }}</span>
                </div>
              </div>
            </template>
            <p v-else class="ollama-note">No usage in the last 14 days.</p>
          </div>
          <p v-else-if="pollStatus === 'error'" class="ollama-error" role="alert">{{ pollError }}</p>
        </div>
      </div>

      <div class="provider-section">
        <div class="setting-title"><div><div class="section-label">Behavior</div><h2>Model parameters</h2><p>JSON object merged into every request — temperature, max tokens, and so on.</p></div></div>
        <FormField label="Model parameters (JSON)" :error="parametersError">
          <textarea v-model="parametersText" class="text-input text-area" rows="5" spellcheck="false" />
        </FormField>
        <label class="checkbox-row"><input v-model="isActive" type="checkbox" /><span>Active provider (routed to next request)</span></label>
      </div>

      <div class="modal-submit-row">
        <button type="button" class="base-button button-secondary" @click="emit('close')">Cancel</button>
        <button type="submit" class="base-button button-primary" :disabled="saving || !name || !baseUrl || !model">{{ saving ? 'Saving…' : provider ? 'Save changes' : 'Create provider' }}</button>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.provider-form { gap: 18px; }
.provider-section { display: grid; gap: 14px; padding-bottom: 18px; border-bottom: 1px dashed var(--color-border); }
.provider-section:last-of-type { padding-bottom: 0; border-bottom: 0; }
.setting-title { display: flex; align-items: start; justify-content: space-between; gap: 16px; }
.setting-title h2 { margin: 4px 0 0; font-size: 1rem; letter-spacing: -.02em; }
.setting-title p { margin: 4px 0 0; color: var(--color-muted); font-size: .8rem; line-height: 1.5; }
.ollama-load { min-height: 38px; }
.ollama-status { display: grid; gap: 8px; margin-top: 2px; padding: 12px 14px; border: 1px solid color-mix(in srgb, #34d399 35%, var(--color-border)); border-radius: 12px; background: color-mix(in srgb, #34d399 7%, var(--color-surface)); font-size: .8rem; }
.ollama-status-row { display: flex; align-items: center; gap: 8px; font-weight: 650; }
.ollama-dot { width: 8px; height: 8px; flex: 0 0 auto; border-radius: 50%; background: #34d399; box-shadow: 0 0 0 3px color-mix(in srgb, #34d399 22%, transparent); }
.ollama-running-title { margin-top: 4px; color: var(--color-muted); font-size: .68rem; font-weight: 720; letter-spacing: .07em; text-transform: uppercase; }
.ollama-running { display: flex; flex-wrap: wrap; align-items: baseline; gap: 4px 12px; }
.ollama-running-name { font-weight: 680; }
.ollama-running-stat { color: var(--color-muted); font-size: .74rem; font-variant-numeric: tabular-nums; }
.ollama-note { margin: 0; color: var(--color-muted); font-size: .76rem; }
.ollama-error { margin: 2px 0 0; padding: 10px 12px; border: 1px solid color-mix(in srgb, var(--color-danger) 35%, var(--color-border)); border-radius: 10px; background: color-mix(in srgb, var(--color-danger) 8%, var(--color-surface)); color: var(--color-danger); font-size: .78rem; line-height: 1.5; }
.poll-key-row { display: flex; flex-wrap: wrap; gap: 4px 12px; align-items: baseline; font-size: .78rem; }
.poll-key-type { font-weight: 720; text-transform: uppercase; letter-spacing: .04em; }
.poll-key-meta { color: var(--color-muted); font-size: .74rem; }
.poll-key-invalid { margin: 2px 0 0; color: var(--color-danger); font-size: .76rem; }
.poll-daily-table { display: grid; gap: 2px; margin-top: 4px; }
.poll-daily-row { display: grid; grid-template-columns: 92px 1fr auto auto; gap: 10px; align-items: baseline; font-size: .74rem; }
.poll-daily-date { color: var(--color-muted); font-variant-numeric: tabular-nums; }
.poll-daily-model { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.poll-daily-stat { color: var(--color-muted); font-variant-numeric: tabular-nums; }
@media (max-width: 480px) { .poll-daily-row { grid-template-columns: 82px 1fr auto; } .poll-daily-row .poll-daily-stat:nth-of-type(2) { display: none; } }
</style>
