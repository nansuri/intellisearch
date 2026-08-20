<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  createMiniApp,
  deleteMiniApp,
  getMyMiniApp,
  updateMiniApp,
  type MiniApp,
  type MiniAppPatch,
} from '../services/api'
import { useToastStore } from '../stores/toast'
import { snippetFor, type MiniAppSource } from '../composables/useMiniAppRunner'
import PageHeader from '../components/PageHeader.vue'
import LoadingSpinner from '../components/LoadingSpinner.vue'
import MiniAppEditor from '../components/apps/MiniAppEditor.vue'
import MiniAppFrame from '../components/apps/MiniAppFrame.vue'
import ConfirmModal from '../components/ConfirmModal.vue'
import FormField from '../components/FormField.vue'

const route = useRoute()
const router = useRouter()
const toast = useToastStore()

const isNew = computed(() => route.params.id === 'new')
const appId = computed(() => (isNew.value ? null : (route.params.id as string)))

const loading = ref(!isNew.value)
const saving = ref(false)
const app = ref<MiniApp | null>(null)

// Form fields
const name = ref('')
const description = ref('')
const icon = ref('')
const html = ref('')
const css = ref('')
const js = ref('')
const visibility = ref<'public' | 'private'>('private')

// Delete confirmation
const showDeleteConfirm = ref(false)
const deleting = ref(false)

// Preview source derived from the editors
const source = computed<MiniAppSource>(() => ({
  html: html.value || snippetFor('html', ''),
  css: css.value || snippetFor('css', ''),
  js: js.value || snippetFor('js', ''),
}))

// Insert starter snippets when fields are empty on first load
const htmlPlaceholder = snippetFor('html', '')
const cssPlaceholder = snippetFor('css', '')
const jsPlaceholder = snippetFor('js', '')

async function loadApp() {
  if (!appId.value) return
  loading.value = true
  try {
    const data = await getMyMiniApp(appId.value)
    app.value = data
    name.value = data.name
    description.value = data.description
    icon.value = data.icon
    html.value = data.html
    css.value = data.css
    js.value = data.js
    visibility.value = data.visibility
  } catch (e) {
    toast.error((e as Error).message || 'Could not load the app.')
    router.push('/apps')
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!name.value.trim()) {
    toast.error('Give your app a name before saving.')
    return
  }
  saving.value = true
  try {
    const input = {
      name: name.value.trim(),
      description: description.value,
      icon: icon.value,
      html: html.value,
      css: css.value,
      js: js.value,
      visibility: visibility.value,
    }
    if (app.value) {
      const updated = await updateMiniApp(app.value.id, input as MiniAppPatch)
      app.value = updated
      toast.success('App saved.')
    } else {
      const created = await createMiniApp(input)
      app.value = created
      toast.success('App created.')
      router.replace({ path: `/apps/${created.id}` })
    }
  } catch (e) {
    toast.error((e as Error).message || 'Could not save.')
  } finally {
    saving.value = false
  }
}

async function toggleVisibility() {
  if (!app.value) return
  const next = visibility.value === 'public' ? 'private' : 'public'
  try {
    const updated = await updateMiniApp(app.value.id, { visibility: next })
    app.value = updated
    visibility.value = next
    toast.success(`App is now ${next}.`)
  } catch (e) {
    toast.error((e as Error).message || 'Could not update visibility.')
  }
}

async function confirmDelete() {
  if (!app.value) return
  deleting.value = true
  try {
    await deleteMiniApp(app.value.id)
    toast.success('App deleted.')
    router.push('/apps')
  } catch (e) {
    toast.error((e as Error).message || 'Could not delete.')
  } finally {
    deleting.value = false
    showDeleteConfirm.value = false
  }
}

function goBack() {
  router.push('/apps')
}

onMounted(() => {
  if (!isNew.value) loadApp()
})

// Apply snippet placeholders when a field is blank
function insertSnippet(field: 'html' | 'css' | 'js') {
  const map = { html, css, js }
  const current = map[field].value
  if (!current.trim()) map[field].value = snippetFor(field, current)
}
</script>

<template>
  <main class="page-shell ide-page">
    <LoadingSpinner v-if="loading" />

    <template v-else>
      <div class="ide-head">
        <PageHeader
          :eyebrow="isNew ? 'New app' : 'Edit app'"
          :title="isNew ? 'Create a mini app' : (app?.name || 'Edit app')"
        >
          <div class="ide-head-actions">
            <button type="button" class="base-button button-secondary" @click="goBack">Back to Studio</button>
            <button type="button" class="base-button button-primary" :disabled="saving || !name.trim()" @click="save">
              {{ saving ? 'Saving…' : (app ? 'Save changes' : 'Create app') }}
            </button>
          </div>
        </PageHeader>
      </div>

      <div class="ide-toolbar">
        <FormField label="Name" class="ide-name-field">
          <input v-model="name" class="text-input" type="text" maxlength="80" placeholder="My mini app" />
        </FormField>
        <FormField label="Description" class="ide-desc-field">
          <input v-model="description" class="text-input" type="text" maxlength="200" placeholder="A short description" />
        </FormField>
        <FormField label="Icon" class="ide-icon-field">
          <input v-model="icon" class="text-input" type="text" maxlength="16" placeholder="🧩" />
        </FormField>
        <div class="ide-vis-group">
          <span class="ide-vis-label">Visibility</span>
          <button
            type="button"
            class="base-button ide-vis-toggle"
            :class="visibility === 'public' ? 'ide-vis--public' : 'ide-vis--private'"
            @click="toggleVisibility"
          >
            {{ visibility === 'public' ? '🌍 Public' : '🔒 Private' }}
          </button>
        </div>
        <button
          v-if="app"
          type="button"
          class="base-button button-danger ide-delete-btn"
          @click="showDeleteConfirm = true"
        >
          Delete
        </button>
      </div>

      <div class="ide-body">
        <div class="ide-editors">
          <div class="ide-editor-group">
            <div class="ide-editor-head">
              <span class="ide-editor-label">HTML</span>
              <button type="button" class="ide-snippet-btn" @click="insertSnippet('html')">Insert sample</button>
            </div>
            <MiniAppEditor v-model="html" label="" language="html" :rows="12" placeholder="<!-- Your HTML -->" />
          </div>
          <div class="ide-editor-group">
            <div class="ide-editor-head">
              <span class="ide-editor-label">CSS</span>
              <button type="button" class="ide-snippet-btn" @click="insertSnippet('css')">Insert sample</button>
            </div>
            <MiniAppEditor v-model="css" label="" language="css" :rows="10" placeholder="/* Your styles */" />
          </div>
          <div class="ide-editor-group">
            <div class="ide-editor-head">
              <span class="ide-editor-label">JavaScript</span>
              <button type="button" class="ide-snippet-btn" @click="insertSnippet('js')">Insert sample</button>
            </div>
            <MiniAppEditor v-model="js" label="" language="js" :rows="12" placeholder="// Your code" />
          </div>
        </div>

        <div class="ide-preview">
          <div class="ide-preview-head">
            <span class="ide-preview-label">Live preview</span>
          </div>
          <div class="ide-preview-frame">
            <MiniAppFrame :source="source" :title="name || 'Preview'" />
          </div>
        </div>
      </div>
    </template>

    <ConfirmModal
      :open="showDeleteConfirm"
      title="Delete this app?"
      message="This action cannot be undone. The app and all its code will be permanently removed."
      :busy="deleting"
      confirm-label="Delete"
      @confirm="confirmDelete"
      @close="showDeleteConfirm = false"
    />
  </main>
</template>

<style scoped>
.ide-page { padding-top: 0; }
.ide-head { margin-top: 28px; }
.ide-head :deep(.admin-page-head) { margin-bottom: 16px; }
.ide-head-actions { display: flex; align-items: center; gap: 12px; }

.ide-toolbar {
  display: flex;
  align-items: flex-end;
  gap: 14px;
  flex-wrap: wrap;
  padding: 16px;
  margin-bottom: 18px;
  border: 1px solid var(--color-border);
  border-radius: 14px;
  background: var(--color-surface);
}
.ide-name-field { flex: 1 1 180px; }
.ide-desc-field { flex: 2 1 260px; }
.ide-icon-field { flex: 0 0 80px; }
.ide-vis-group { display: flex; flex-direction: column; gap: 4px; }
.ide-vis-label {
  color: var(--color-muted);
  font-size: .68rem;
  font-weight: 720;
  letter-spacing: .06em;
  text-transform: uppercase;
}
.ide-vis-toggle {
  white-space: nowrap;
  font-size: .82rem;
  font-weight: 650;
}
.ide-vis--public {
  background: color-mix(in srgb, #2e9d5b 14%, var(--color-surface));
  color: #1f7a46;
  border-color: color-mix(in srgb, #2e9d5b 30%, var(--color-border));
}
.ide-vis--private {
  background: var(--color-surface-subtle);
  color: var(--color-muted);
  border-color: var(--color-border);
}
.ide-delete-btn { margin-left: auto; }

.ide-body {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  min-height: 480px;
}

.ide-editors {
  display: grid;
  gap: 12px;
  min-width: 0;
}
.ide-editor-group {
  display: grid;
  gap: 4px;
  min-width: 0;
}
.ide-editor-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.ide-editor-label {
  color: var(--color-muted);
  font-size: .7rem;
  font-weight: 720;
  letter-spacing: .06em;
  text-transform: uppercase;
}
.ide-snippet-btn {
  padding: 2px 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--color-primary);
  font-size: .72rem;
  font-weight: 620;
  cursor: pointer;
}
.ide-snippet-btn:hover { background: color-mix(in srgb, var(--color-primary) 10%, transparent); }

.ide-preview {
  display: grid;
  grid-template-rows: auto 1fr;
  border: 1px solid var(--color-border);
  border-radius: 14px;
  background: var(--color-surface);
  overflow: hidden;
}
.ide-preview-head {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid var(--color-border);
}
.ide-preview-label {
  color: var(--color-muted);
  font-size: .7rem;
  font-weight: 720;
  letter-spacing: .06em;
  text-transform: uppercase;
}
.ide-preview-frame {
  width: 100%;
  height: 100%;
  min-height: 400px;
}

.button-danger {
  border-color: color-mix(in srgb, var(--color-danger, #d94848) 40%, var(--color-border));
  color: var(--color-danger, #d94848);
}
.button-danger:hover {
  background: color-mix(in srgb, var(--color-danger, #d94848) 10%, var(--color-surface));
}

@media (max-width: 900px) {
  .ide-body { grid-template-columns: 1fr; }
  .ide-preview-frame { min-height: 320px; }
}
</style>
