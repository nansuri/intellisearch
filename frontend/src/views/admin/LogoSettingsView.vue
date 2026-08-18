<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { deleteLogo, getSiteSettings, uploadLogo } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import { useSiteStore } from '../../stores/site'
import PageHeader from '../../components/PageHeader.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import ConfirmModal from '../../components/ConfirmModal.vue'
import BrandPreview from '../../components/admin/BrandPreview.vue'

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
    <PageHeader eyebrow="Branding" title="Logo" description="The square logo shown in the header, the admin sidebar, and on the main page." />
    <section v-if="loading" class="admin-loading"><LoadingSpinner /></section>
    <template v-else>
      <section class="admin-card">
        <div class="mini-head">
          <h2>Current logo</h2>
          <button v-if="logoUrl" class="table-action table-action--danger" @click="pendingDelete = true">Remove logo</button>
        </div>
        <div class="logo-current">
          <img v-if="logoUrl" :src="logoUrl" alt="Current logo" class="logo-current-img" />
          <span v-else class="brand-mark logo-current-fallback">{{ (site.settings?.siteName || 'AI').slice(0, 2).toUpperCase() }}</span>
          <div class="logo-current-copy">
            <strong>{{ logoUrl ? 'Custom logo' : 'Default initials mark' }}</strong>
            <span class="form-hint">{{ logoUrl ? 'This image is served from this deployment and replaces the initials everywhere.' : 'No logo yet — the header shows the site initials until you upload one.' }}</span>
          </div>
        </div>
      </section>

      <section class="admin-card">
        <div class="setting-title"><div><div class="section-label">Upload</div><h2>Replace logo</h2><p>PNG, JPG, WebP or GIF · up to 2 MB. Re-uploading gives the file a fresh URL, so browsers never serve a stale cached copy.</p></div></div>
        <div
          class="dropzone" :class="{ 'dropzone--over': dragging }"
          @dragover.prevent="dragging = true" @dragleave.prevent="dragging = false" @drop.prevent="onDrop"
          @click="fileInput?.click()"
        >
          <input ref="fileInput" type="file" accept="image/png,image/jpeg,image/webp,image/gif" hidden @change="pick()" />
          <strong>{{ saving ? 'Uploading…' : 'Choose or drop a logo' }}</strong>
          <span>Square images look best — the logo is shown at a small size in the header</span>
        </div>
      </section>

      <section class="admin-card">
        <div class="mini-head"><h2>How it looks</h2><span class="admin-site-link">As shown in the site header</span></div>
        <BrandPreview :site-name="site.settings?.siteName || ''" :tagline="site.settings?.tagline || ''" :logo-url="logoUrl" />
      </section>
    </template>
    <ConfirmModal :open="pendingDelete" title="Remove logo" message="Remove the current logo? The main page and header fall back to the default site-initials mark." confirm-label="Remove logo" :busy="busy" @close="pendingDelete = false" @confirm="confirmDelete" />
  </div>
</template>

<style scoped>
.logo-current { display: flex; align-items: center; gap: 16px; margin-top: 6px; }
.logo-current-img { width: 72px; height: 72px; flex: 0 0 auto; border: 1px solid var(--color-border); border-radius: 16px; object-fit: contain; background: var(--color-bg); }
.logo-current-fallback { width: 72px; height: 72px; flex: 0 0 auto; border-radius: 16px; font-size: 1.2rem; background: linear-gradient(135deg, var(--color-primary), color-mix(in srgb, var(--color-primary) 55%, #8b6dff)); color: var(--color-primary-contrast); }
.logo-current-copy { display: grid; gap: 3px; }
.logo-current-copy strong { font-size: .95rem; letter-spacing: -.02em; }
.logo-current-copy .form-hint { max-width: 420px; }
.setting-title { display: flex; align-items: start; justify-content: space-between; gap: 16px; }
.setting-title h2 { margin: 4px 0 0; font-size: 1rem; letter-spacing: -.02em; }
.setting-title p { margin: 4px 0 0; color: var(--color-muted); font-size: .8rem; line-height: 1.5; }
</style>
