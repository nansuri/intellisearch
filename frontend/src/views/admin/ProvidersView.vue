<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { createProvider, deleteProvider, listProviders, updateProvider, type Provider } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import PageHeader from '../../components/PageHeader.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import EmptyState from '../../components/EmptyState.vue'
import ConfirmModal from '../../components/ConfirmModal.vue'
import AdminIcon from '../../components/admin/AdminIcon.vue'
import ProviderFormModal from './ProviderFormModal.vue'

const toast = useToastStore()
const providers = ref<Provider[]>([]); const loading = ref(true)
const editing = ref<Provider | null>(null); const modalOpen = ref(false)
const pendingDelete = ref<Provider | null>(null); const busy = ref(false)

async function load() {
  loading.value = true
  try { providers.value = (await listProviders()).providers } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
async function save(payload: Parameters<typeof createProvider>[0]) {
  try {
    if (editing.value) await updateProvider(editing.value.id, payload)
    else await createProvider(payload)
    modalOpen.value = false; toast.success(editing.value ? 'Provider updated.' : 'Provider created.')
    load()
  } catch (e) { toast.error((e as Error).message) }
}
async function confirmDelete() {
  if (!pendingDelete.value) return
  busy.value = true
  try { await deleteProvider(pendingDelete.value.id); toast.success('Provider removed.'); pendingDelete.value = null; load() }
  catch (e) { toast.error((e as Error).message) } finally { busy.value = false }
}
async function setActive(p: Provider) {
  try { await updateProvider(p.id, { name: p.name, providerType: p.providerType, baseUrl: p.baseUrl, model: p.model, parameters: p.parameters, isActive: true }); toast.success(`${p.name} is now active.`); load() }
  catch (e) { toast.error((e as Error).message) }
}
const typeLabel: Record<string, { label: string; tone: string }> = {
  ollama: { label: 'Ollama', tone: 'success' },
  openai_compatible: { label: 'OpenAI-compatible', tone: 'accent' },
  pollinations: { label: 'Pollinations.ai', tone: 'accent' },
  huggingface: { label: 'Hugging Face', tone: 'accent' },
}
const providerIcon: Record<string, string> = {
  ollama: 'robot',
  openai_compatible: 'ai',
  pollinations: 'branding',
  huggingface: 'settings',
}
const activeProvider = computed(() => providers.value.find((p) => p.isActive) || null)
onMounted(load)
</script>
<template>
  <div>
    <PageHeader eyebrow="AI providers" title="Providers" description="Backends serving answers. Exactly one provider is active at a time — all requests route through it.">
      <button class="base-button button-primary" @click="editing = null; modalOpen = true">New provider</button>
    </PageHeader>
    <section v-if="loading" class="admin-loading"><LoadingSpinner /></section>
    <template v-else>
      <section class="provider-summary">
        <div class="latency-cell"><span>Providers</span><strong>{{ providers.length }}</strong></div>
        <div class="latency-cell"><span>Active provider</span><strong class="provider-summary-name">{{ activeProvider?.name || 'None' }}</strong></div>
        <div class="latency-cell"><span>Active model</span><strong class="provider-summary-name">{{ activeProvider?.model || '—' }}</strong></div>
        <div class="latency-cell"><span>Active type</span><strong>{{ activeProvider ? (typeLabel[activeProvider.providerType]?.label || activeProvider.providerType) : '—' }}</strong></div>
      </section>
      <section v-if="providers.length" class="provider-grid">
        <article v-for="p in providers" :key="p.id" class="provider-card" :class="{ 'provider-card--active': p.isActive }">
          <div class="provider-card-head">
            <span class="provider-card-icon" aria-hidden="true"><AdminIcon :name="providerIcon[p.providerType] || 'settings'" :size="18" /></span>
            <div class="provider-card-title"><h3>{{ p.name }}</h3><span class="provider-card-type">{{ typeLabel[p.providerType]?.label || p.providerType }}</span></div>
            <span v-if="p.isActive" class="status-badge status-badge--success">Active</span>
            <span v-else class="status-badge">Standby</span>
          </div>
          <dl class="provider-meta-list">
            <div><dt>Model</dt><dd>{{ p.model }}</dd></div>
            <div><dt>Base URL</dt><dd class="provider-meta-url">{{ p.baseUrl }}</dd></div>
            <div v-if="p.parameters && p.parameters.temperature !== undefined"><dt>Temperature</dt><dd>{{ p.parameters.temperature }}</dd></div>
          </dl>
          <div class="provider-actions">
            <button v-if="!p.isActive" class="base-button button-secondary provider-set-active" @click="setActive(p)">Set active</button>
            <button class="table-action" @click="editing = p; modalOpen = true">Edit</button>
            <button class="table-action table-action--danger" @click="pendingDelete = p">Delete</button>
          </div>
        </article>
      </section>
      <EmptyState v-else title="No providers yet" message="Create a provider to start answering questions.">
        <button class="base-button button-secondary" @click="editing = null; modalOpen = true">Add your first provider</button>
      </EmptyState>
    </template>
    <ProviderFormModal :open="modalOpen" :provider="editing" @close="modalOpen = false" @save="save" />
    <ConfirmModal v-if="pendingDelete" :open="true" title="Delete provider" :message="`Delete the '${pendingDelete.name}' provider? Keep at least one provider configured.`" confirm-label="Delete" :busy="busy" @close="pendingDelete = null" @confirm="confirmDelete" />
  </div>
</template>

<style scoped>
.provider-summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin-bottom: 22px; }
.provider-summary .latency-cell strong { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.provider-summary-name { font-size: 1.02rem; }
.provider-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 14px; }
.provider-card { position: relative; display: grid; gap: 14px; padding: 18px; border: 1px solid var(--color-border); border-radius: 16px; background: linear-gradient(180deg, color-mix(in srgb, var(--color-surface) 88%, var(--color-primary) 2%), var(--color-surface)); box-shadow: 0 14px 36px var(--color-shadow); transition: border-color .15s ease, transform .15s ease; }
.provider-card:hover { transform: translateY(-1px); }
.provider-card--active { border-color: color-mix(in srgb, var(--color-primary) 55%, var(--color-border)); }
.provider-card--active::before { content: ''; position: absolute; inset: 0 0 auto 0; height: 2px; border-radius: 16px 16px 0 0; background: linear-gradient(90deg, transparent, var(--color-primary), transparent); opacity: .7; }
.provider-card-head { display: flex; align-items: center; gap: 11px; min-width: 0; }
.provider-card-icon { display: grid; place-items: center; width: 36px; height: 36px; flex: 0 0 auto; border-radius: 10px; background: color-mix(in srgb, var(--color-primary) 14%, var(--color-surface)); color: var(--color-primary); }
.provider-card-title { display: grid; gap: 1px; min-width: 0; flex: 1; }
.provider-card-title h3 { margin: 0; font-size: 1rem; letter-spacing: -.02em; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.provider-card-type { color: var(--color-muted); font-size: .7rem; font-weight: 700; letter-spacing: .06em; text-transform: uppercase; }
.provider-meta-list { display: grid; gap: 7px; margin: 0; padding: 12px 14px; border: 1px solid var(--color-border); border-radius: 12px; background: color-mix(in srgb, var(--color-bg) 55%, transparent); }
.provider-meta-list div { display: grid; grid-template-columns: 92px minmax(0, 1fr); gap: 10px; align-items: baseline; }
.provider-meta-list dt { color: var(--color-muted); font-size: .72rem; font-weight: 700; letter-spacing: .05em; text-transform: uppercase; }
.provider-meta-list dd { margin: 0; font-size: .82rem; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.provider-meta-url { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: .74rem; font-weight: 500; }
.provider-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; padding-top: 12px; border-top: 1px solid var(--color-border); }
.provider-set-active { margin-right: auto; min-height: 34px; padding: 0 12px; font-size: .76rem; }
@media (max-width: 640px) { .provider-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
