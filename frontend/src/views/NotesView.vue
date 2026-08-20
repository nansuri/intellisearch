<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { createNote, deleteNote, listNotes, updateNote, type AskMode, type Note } from '../services/api'
import { useToastStore } from '../stores/toast'
import PageHeader from '../components/PageHeader.vue'
import BaseModal from '../components/BaseModal.vue'
import ConfirmModal from '../components/ConfirmModal.vue'
import FormField from '../components/FormField.vue'
import LoadingSpinner from '../components/LoadingSpinner.vue'
import EmptyState from '../components/EmptyState.vue'
import RadioSwitch from '../components/RadioSwitch.vue'
import AppHeader from '../components/AppHeader.vue'
import AskBox from '../components/AskBox.vue'
import MarkdownView from '../components/MarkdownView.vue'
import { relativeTime } from '../utils/format'

const router = useRouter()
const toast = useToastStore()
const notes = ref<Note[]>([])
const loading = ref(true)
const saving = ref(false)
const editorOpen = ref(false)
const editing = ref<Note | null>(null)
const previewing = ref<Note | null>(null)
const title = ref('')
const content = ref('')
const deleting = ref(false)
const confirm = ref<Note | null>(null)

// Cards vs list view, remembered per device.
const VIEW_KEY = 'notes.view'
const viewMode = ref<'grid' | 'list'>((localStorage.getItem(VIEW_KEY) as 'grid' | 'list') || 'grid')
watch(viewMode, (mode) => localStorage.setItem(VIEW_KEY, mode))

function onAsk(question: string, mode: AskMode) {
  router.push({ path: '/search', query: { q: question, mode } })
}

async function load() {
  loading.value = true
  try { notes.value = (await listNotes()).items } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
onMounted(load)

function openNew() {
  editing.value = null
  title.value = ''
  content.value = ''
  editorOpen.value = true
}
function openEdit(note: Note) {
  editing.value = note
  title.value = note.title
  content.value = note.content
  editorOpen.value = true
}

async function save() {
  if (!title.value.trim() || !content.value.trim()) return
  saving.value = true
  try {
    if (editing.value) {
      await updateNote(editing.value.id, { title: title.value.trim(), content: content.value.trim() })
      toast.success('Note updated.')
    } else {
      await createNote({ title: title.value.trim(), content: content.value.trim() })
      toast.success('Note saved.')
    }
    editorOpen.value = false
    await load()
  } catch (e) { toast.error((e as Error).message) } finally { saving.value = false }
}

function requestDelete(note: Note) { confirm.value = note }

async function runDelete() {
  if (!confirm.value) return
  deleting.value = true
  try {
    await deleteNote(confirm.value.id)
    toast.success('Note deleted.')
    confirm.value = null
    await load()
  } catch (e) {
    toast.error((e as Error).message)
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <main class="page-shell notes-page">
    <AppHeader compact>
      <template #center>
        <AskBox variant="google" placeholder="Ask a question, explore an idea…" @submit="onAsk" />
      </template>
    </AppHeader>

    <div class="notes-head">
      <PageHeader eyebrow="Apps" title="Notes" description="Save summaries and ideas. Anything you save from a search is linked back to it.">
        <div class="notes-head-actions">
          <RadioSwitch
            v-model="viewMode"
            :options="[{ value: 'grid', label: 'Cards' }, { value: 'list', label: 'List' }]"
            size="sm"
            variant="pill"
            aria-label="Notes view"
          />
          <button type="button" class="base-button button-primary" @click="openNew">+ New note</button>
        </div>
      </PageHeader>
    </div>

    <div v-if="loading" class="notes-loading"><LoadingSpinner /></div>

    <section v-else-if="notes.length" class="notes-layout" :class="`notes-layout--${viewMode}`">
      <article v-for="note in notes" :key="note.id" class="note-card">
        <div class="note-card-head">
          <button type="button" class="note-card-title" :title="`Preview: ${note.title}`" @click="previewing = note">{{ note.title }}</button>
          <div class="note-card-actions">
            <button type="button" class="note-action" title="Preview" @click="previewing = note">Preview</button>
            <button type="button" class="note-action" title="Edit" @click="openEdit(note)">Edit</button>
            <button type="button" class="note-action note-action--danger" title="Delete" @click="requestDelete(note)">Delete</button>
          </div>
        </div>
        <div class="note-content">
          <MarkdownView v-if="note.content" :content="note.content" />
          <p v-else class="note-content-empty">This note has no content.</p>
        </div>
        <div class="note-card-meta">
          <span v-if="note.sourceQuery" class="note-source">From search: “{{ note.sourceQuery }}”</span>
          <span>{{ relativeTime(note.createdAt) }}</span>
        </div>
      </article>
    </section>

    <EmptyState v-else icon="📝" title="No notes yet" message="Save a search summary, or write your own note — anything you add shows up here.">
      <button type="button" class="base-button button-primary" @click="openNew">Write your first note</button>
    </EmptyState>

    <BaseModal :open="Boolean(previewing)" :title="previewing?.title || 'Note'" size="lg" @close="previewing = null">
      <div class="note-preview-body">
        <div v-if="previewing" class="note-preview-meta">
          <span v-if="previewing.sourceQuery">From search: “{{ previewing.sourceQuery }}”</span>
          <span>{{ relativeTime(previewing.createdAt) }}</span>
        </div>
        <MarkdownView v-if="previewing?.content" :content="previewing.content" />
        <p v-else class="note-preview-empty">This note has no content.</p>
      </div>
    </BaseModal>

    <ConfirmModal
      v-if="confirm"
      :open="true"
      title="Delete note"
      :message="`Delete “${confirm.title}” permanently? This cannot be undone.`"
      confirm-label="Delete"
      :busy="deleting"
      @close="confirm = null"
      @confirm="runDelete"
    />

    <BaseModal :open="editorOpen" :title="editing ? 'Edit note' : 'New note'" :busy="saving" @close="editorOpen = false">
      <form class="notes-form" @submit.prevent="save">
        <FormField label="Title">
          <input v-model="title" class="text-input" required maxlength="120" placeholder="Note title" />
        </FormField>
        <FormField label="Content">
          <textarea v-model="content" class="text-input text-area notes-textarea" required rows="8" placeholder="Write or paste your note…" />
        </FormField>
      </form>
      <template #footer>
        <button type="button" class="base-button button-secondary" @click="editorOpen = false">Cancel</button>
        <button type="submit" class="base-button button-primary" :disabled="saving || !title.trim() || !content.trim()" @click="save">
          {{ saving ? 'Saving…' : editing ? 'Save changes' : 'Save note' }}
        </button>
      </template>
    </BaseModal>
  </main>
</template>

<style scoped>
.notes-page { padding-top: 0; }
.notes-head { margin-top: 28px; }
.notes-head :deep(.admin-page-head) { margin-bottom: 22px; }
.notes-head-actions { display: flex; align-items: center; gap: 12px; }
.notes-loading { display: grid; place-items: center; padding: 60px 0; }
.notes-layout { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px; }
.notes-layout--list { grid-template-columns: 1fr; max-width: 860px; margin-inline: auto; }
.note-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  padding: 20px;
  border: 1px solid var(--color-border);
  border-radius: 16px;
  background: var(--color-surface);
  box-shadow: 0 10px 30px var(--color-shadow);
}
/* List view reads like a document index: row layout, tighter preview. */
.notes-layout--list .note-card { flex-direction: row; flex-wrap: wrap; gap: 8px 16px; padding: 16px 20px; }
.notes-layout--list .note-card-head { flex: 1 1 100%; }
.notes-layout--list .note-content { flex: 1 1 60%; }
.notes-layout--list .note-card-meta { margin-top: 0; padding-top: 0; border-top: 0; }
.note-card-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.note-card-title {
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--color-text);
  font-size: 1.02rem;
  font-weight: 720;
  letter-spacing: -.02em;
  text-align: left;
  cursor: pointer;
}
.note-card-title:hover { color: var(--color-primary); text-decoration: underline; }
.note-card-actions { display: flex; gap: 4px; flex: 0 0 auto; }
.note-action {
  padding: 4px 8px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--color-muted);
  font-size: .72rem;
  font-weight: 680;
  cursor: pointer;
}
.note-action:hover { background: var(--color-surface-subtle); color: var(--color-text); }
.note-action--danger:hover { color: var(--color-danger); }
.note-content { margin: 0; min-width: 0; }
.note-content-empty { margin: 0; color: var(--color-muted); font-size: .86rem; }
/* Markdown preview on the card is clamped to a few lines; the modal shows the
   full note. Tables scroll instead of blowing out the card. */
.note-content :deep(.markdown) {
  margin: 0;
  color: var(--color-muted);
  font-size: .86rem;
  line-height: 1.6;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
  overflow-wrap: anywhere;
}
.note-content :deep(.markdown p) { margin: 0 0 .45em; }
.note-content :deep(.markdown p:last-child) { margin-bottom: 0; }
.note-content :deep(.markdown table) { display: block; overflow-x: auto; }
.notes-layout--list .note-content :deep(.markdown) { -webkit-line-clamp: 2; }
.note-card-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  margin-top: auto;
  padding-top: 12px;
  border-top: 1px solid var(--color-border);
  color: var(--color-muted);
  font-size: .74rem;
}
.note-source { color: var(--color-primary); font-weight: 620; }
.notes-form { display: grid; gap: 14px; }
.notes-textarea { min-height: 180px; resize: vertical; line-height: 1.6; }
.note-preview-body { display: grid; gap: 12px; min-width: 0; }
.note-preview-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border);
  color: var(--color-muted);
  font-size: .76rem;
}
.note-preview-meta span:first-child { color: var(--color-primary); font-weight: 620; }
.note-preview-empty { margin: 0; color: var(--color-muted); font-size: .9rem; }
/* Markdown inside the modal must wrap/scroll within the dialog instead of
   blowing it out (long URLs, wide tables). */
.note-preview-body :deep(.markdown) { overflow-wrap: anywhere; }
.note-preview-body :deep(.markdown table) { display: block; overflow-x: auto; }
@media (max-width: 640px) {
  .notes-layout { grid-template-columns: 1fr; }
  .notes-head-actions { flex-direction: column; align-items: stretch; }
}
</style>
