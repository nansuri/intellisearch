<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createProvider, deleteProvider, listProviders, updateProvider, type Provider } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import PageHeader from '../../components/PageHeader.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import EmptyState from '../../components/EmptyState.vue'
import ConfirmModal from '../../components/ConfirmModal.vue'
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
onMounted(load)
</script>
<template>
  <div>
    <PageHeader eyebrow="AI providers" title="Providers" description="Backends serving answers. Exactly one provider is active at a time — all requests route through it.">
      <button class="base-button button-primary" @click="editing = null; modalOpen = true">New provider</button>
    </PageHeader>
    <section class="admin-card">
      <div v-if="loading" class="admin-loading"><LoadingSpinner /></div>
      <div v-else-if="providers.length" class="provider-grid">
        <article v-for="p in providers" :key="p.id" class="provider-card">
          <div class="provider-row">
            <StatusBadge :value="p.providerType" :options="typeLabel" />
            <span v-if="p.isActive" class="status-badge status-badge--success">Active</span>
            <span v-else class="status-badge">Standby</span>
          </div>
          <h3>{{ p.name }}</h3>
          <p class="provider-model">{{ p.model }}</p>
          <p class="provider-url">{{ p.baseUrl }}</p>
          <p v-if="p.parameters && p.parameters.temperature !== undefined" class="provider-meta">temperature {{ p.parameters.temperature }}</p>
          <div class="provider-actions">
            <button v-if="!p.isActive" class="table-action" @click="setActive(p)">Set active</button>
            <button class="table-action" @click="editing = p; modalOpen = true">Edit</button>
            <button class="table-action table-action--danger" @click="pendingDelete = p">Delete</button>
          </div>
        </article>
      </div>
      <EmptyState v-else title="No providers yet" message="Create a provider to start answering questions.">
        <button class="base-button button-secondary" @click="editing = null; modalOpen = true">Add your first provider</button>
      </EmptyState>
    </section>
    <ProviderFormModal :open="modalOpen" :provider="editing" @close="modalOpen = false" @save="save" />
    <ConfirmModal v-if="pendingDelete" :open="true" title="Delete provider" :message="`Delete the '${pendingDelete.name}' provider? Keep at least one provider configured.`" confirm-label="Delete" :busy="busy" @close="pendingDelete = null" @confirm="confirmDelete" />
  </div>
</template>