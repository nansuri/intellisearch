<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { deleteFavicon, getSiteSettings, updateSiteSettings, uploadFavicon } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import { useSiteStore } from '../../stores/site'
import PageHeader from '../../components/PageHeader.vue'
import FormField from '../../components/FormField.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import ConfirmModal from '../../components/ConfirmModal.vue'
import BrandPreview from '../../components/admin/BrandPreview.vue'

const toast = useToastStore()
const site = useSiteStore()
const loading = ref(true); const saving = ref(false)
const name = ref(''); const tagline = ref(''); const copyright = ref('')
const logoUrl = ref<string | null>(null)
const faviconUrl = ref<string | null>(null)
const faviconInput = ref<HTMLInputElement | null>(null)
const faviconSaving = ref(false); const faviconDragging = ref(false)
const pendingFaviconDelete = ref(false); const faviconBusy = ref(false)

async function load() {
  loading.value = true
  try {
    const s = await getSiteSettings()
    name.value = s.siteName; tagline.value = s.tagline || ''; copyright.value = s.copyright || ''; logoUrl.value = s.logoUrl; faviconUrl.value = s.faviconUrl
  } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
async function save() {
  if (!name.value.trim()) return
  saving.value = true
  try {
    const updated = await updateSiteSettings({ siteName: name.value.trim(), tagline: tagline.value.trim() || null, copyright: copyright.value.trim() || null })
    site.settings = updated
    toast.success('Identity updated — the public page updates immediately.')
  } catch (e) { toast.error((e as Error).message) } finally { saving.value = false }
}
async function pickFavicon(file?: File) {
  const f = file || faviconInput.value?.files?.[0]
  if (!f) return
  faviconSaving.value = true
  try {
    const result = await uploadFavicon(f)
    faviconUrl.value = result.faviconUrl
    site.settings = { ...site.settings, faviconUrl: result.faviconUrl } as typeof site.settings
    toast.success('Favicon uploaded — the browser tab updates immediately.')
  } catch (e) { toast.error((e as Error).message) } finally { faviconSaving.value = false; if (faviconInput.value) faviconInput.value.value = '' }
}
async function confirmFaviconDelete() {
  faviconBusy.value = true
  try {
    await deleteFavicon()
    faviconUrl.value = null
    site.settings = { ...site.settings, faviconUrl: null } as typeof site.settings
    toast.success('Favicon removed — the default mark is used again.')
    pendingFaviconDelete.value = false
  } catch (e) { toast.error((e as Error).message) } finally { faviconBusy.value = false }
}
function onDrop(e: DragEvent) { faviconDragging.value = false; pickFavicon(e.dataTransfer?.files?.[0]) }
onMounted(load)
</script>
<template>
  <div>
    <PageHeader eyebrow="Branding" title="Identity" description="The site name, tagline, and browser-tab favicon shown across the public pages." />
    <section v-if="loading" class="admin-loading"><LoadingSpinner /></section>
    <template v-else>
      <section class="admin-card">
        <div class="mini-head"><h2>Live preview</h2><span class="admin-site-link">As shown in the site header</span></div>
        <BrandPreview :site-name="name" :tagline="tagline" :logo-url="logoUrl" />
        <p class="preview-note">The preview updates as you type — upload or replace the logo under Branding → Logo.</p>
      </section>

      <section class="admin-card admin-card--form">
        <div class="setting-title"><div><div class="section-label">Site identity</div><h2>Name &amp; tagline</h2><p>Shown in the header, the sidebar, and as the browser-tab title.</p></div></div>
        <FormField label="Site name" :error="name ? '' : 'Site name is required'">
          <input v-model="name" class="text-input" placeholder="My AI" />
        </FormField>
        <FormField label="Tagline" hint="One short line under the name">
          <input v-model="tagline" class="text-input" placeholder="Research, answered." />
        </FormField>
        <FormField label="Copyright" hint="The legal line in the site footer — leave blank to use the site name. Rendered as © year + text.">
          <input v-model="copyright" class="text-input" maxlength="80" placeholder="Acme Search" />
        </FormField>
        <div class="modal-submit-row">
          <button class="base-button button-primary" :disabled="saving || !name.trim()" @click="save">{{ saving ? 'Saving…' : 'Save identity' }}</button>
        </div>
      </section>

      <section class="admin-card">
        <div class="setting-title"><div><div class="section-label">Browser tab</div><h2>Favicon</h2><p>The small icon shown next to the site name in the browser tab. PNG works best (recommended 64×64 or larger).</p></div></div>
        <div class="favicon-row">
          <img v-if="faviconUrl" :src="faviconUrl" alt="Current favicon" class="favicon-preview" />
          <span v-else class="favicon-preview favicon-preview--default"><img src="/favicon.svg" alt="Default favicon" /></span>
          <div class="favicon-copy"><strong>{{ faviconUrl ? 'Custom favicon' : 'Default favicon' }}</strong><span class="form-hint">{{ faviconUrl ? 'The browser tab shows this image.' : 'No custom favicon — the default mark is used.' }}</span></div>
        </div>
        <div
          class="dropzone" :class="{ 'dropzone--over': faviconDragging }"
          @dragover.prevent="faviconDragging = true" @dragleave.prevent="faviconDragging = false" @drop.prevent="onDrop"
          @click="faviconInput?.click()"
        >
          <input ref="faviconInput" type="file" accept="image/png,image/jpeg,image/webp,image/gif" hidden @change="pickFavicon()" />
          <strong>{{ faviconSaving ? 'Uploading…' : 'Choose or drop a favicon' }}</strong>
          <span>PNG, JPG, WebP or GIF · up to 2 MB · square images look best</span>
        </div>
        <div v-if="faviconUrl" class="modal-submit-row">
          <button class="base-button button-danger" @click="pendingFaviconDelete = true">Remove favicon</button>
        </div>
      </section>
    </template>

    <ConfirmModal :open="pendingFaviconDelete" title="Remove favicon" message="Remove the custom favicon? The browser tab falls back to the default mark." confirm-label="Remove favicon" :busy="faviconBusy" @close="pendingFaviconDelete = false" @confirm="confirmFaviconDelete" />
  </div>
</template>

<style scoped>
.preview-note { margin: 12px 0 0; color: var(--color-muted); font-size: .76rem; line-height: 1.5; }
.setting-title { display: flex; align-items: start; justify-content: space-between; gap: 16px; }
.setting-title h2 { margin: 4px 0 0; font-size: 1rem; letter-spacing: -.02em; }
.setting-title p { margin: 4px 0 0; color: var(--color-muted); font-size: .8rem; line-height: 1.5; }
.favicon-row { display: flex; align-items: center; gap: 14px; margin-top: 16px; }
.favicon-preview { width: 48px; height: 48px; flex: 0 0 auto; border: 1px solid var(--color-border); border-radius: 12px; object-fit: contain; background: var(--color-bg); }
.favicon-preview--default { display: grid; place-items: center; background: var(--color-bg); }
.favicon-preview--default img { width: 28px; height: 28px; }
.favicon-copy { display: grid; gap: 2px; }
.favicon-copy strong { font-size: .9rem; }
</style>
