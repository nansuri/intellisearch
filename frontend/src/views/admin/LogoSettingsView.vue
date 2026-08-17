<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { deleteLogo, getSiteSettings, uploadLogo } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import { useSiteStore } from '../../stores/site'
import PageHeader from '../../components/PageHeader.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import ConfirmModal from '../../components/ConfirmModal.vue'

const toast = useToastStore()
const site = useSiteStore()
const loading = ref(true); const saving = ref(false); const dragging = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const logoUrl = ref<string | null>(null)
const pendingDelete = ref(false); const busy = ref(false)

async function load() {
  loading.value = true
  try { logoUrl.value = (await getSiteSettings()).logoUrl } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
async function pick(file?: File) {
  const f = file || fileInput.value?.files?.[0]
  if (!f) return
  saving.value = true
  try {
    const result = await uploadLogo(f)
    logoUrl.value = result.logoUrl; site.settings = { ...site.settings, logoUrl: result.logoUrl } as typeof site.settings
    toast.success('Logo uploaded — the header updates immediately.')
  } catch (e) { toast.error((e as Error).message) } finally { saving.value = false; if (fileInput.value) fileInput.value.value = '' }
}
async function confirmDelete() {
  busy.value = true
  try {
    await deleteLogo()
    logoUrl.value = null; site.settings = { ...site.settings, logoUrl: null } as typeof site.settings
    toast.success('Logo removed — the main page uses the default mark.')
    pendingDelete.value = false
  } catch (e) { toast.error((e as Error).message) } finally { busy.value = false }
}
function onDrop(e: DragEvent) { dragging.value = false; pick(e.dataTransfer?.files?.[0]) }
onMounted(load)
</script>
<template>
  <div>
    <PageHeader eyebrow="Branding" title="Logo" description="Square logo shown in the top-left header and on the main page." />
    <section v-if="loading" class="admin-loading"><LoadingSpinner /></section>
    <section v-else class="admin-card admin-card--form">
      <div class="logo-preview"><img v-if="logoUrl" :src="logoUrl" alt="Current logo" class="logo-preview-img" /><span v-else class="brand-mark">{{ (site.settings?.siteName || 'AI').slice(0, 2).toUpperCase() }}</span><div v-if="logoUrl"><strong>Current logo</strong><span class="form-hint">The header and main page show this image.</span></div></div>
      <div
        class="dropzone" :class="{ 'dropzone--over': dragging }"
        @dragover.prevent="dragging = true" @dragleave.prevent="dragging = false" @drop.prevent="onDrop"
        @click="fileInput?.click()"
      >
        <input ref="fileInput" type="file" accept="image/png,image/jpeg,image/webp,image/gif" hidden @change="pick()" />
        <strong>{{ saving ? 'Uploading…' : 'Choose or drop a logo' }}</strong>
        <span>PNG, JPG, WebP or GIF · up to 2 MB</span>
      </div>
      <p class="form-hint">The default fallback shows the site initials until a logo is provided. Logos are served from this deployment.</p>
      <div v-if="logoUrl" class="modal-submit-row">
        <button class="base-button button-danger" @click="pendingDelete = true">Remove logo</button>
      </div>
    </section>
    <ConfirmModal :open="pendingDelete" title="Remove logo" message="Remove the current logo? The main page and header fall back to the default site-initials mark." confirm-label="Remove logo" :busy="busy" @close="pendingDelete = false" @confirm="confirmDelete" />
  </div>
</template>