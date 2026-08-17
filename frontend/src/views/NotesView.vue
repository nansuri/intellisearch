<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createNote, deleteNote, listNotes, updateNote, type Note } from '../services/api'
import { useToastStore } from '../stores/toast'
import PageHeader from '../components/PageHeader.vue'
import BaseModal from '../components/BaseModal.vue'
import FormField from '../components/FormField.vue'
import LoadingSpinner from '../components/LoadingSpinner.vue'
import EmptyState from '../components/EmptyState.vue'
import { relativeTime } from '../utils/format'

const toast = useToastStore()
const notes = ref<Note[]>([])
const loading = ref(true)
const saving = ref(false)
const editorOpen = ref(false)
const editing = ref<Note | null>(null)
const title = ref('')
const content = ref('')
const confirmingId = ref<number | null>(null)

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

async function remove(note: Note) {
  if (confirmingId.value === note.id) {
    confirmingId.value = null
    try {
      await deleteNote(note.id)
      toast.success('Note deleted.')
      await load()
    } catch (e) { toast.error((e as Error).message) }
    return
  }
  confirmingId.value = note.id
  setTimeout(() => { if (confirmingId.value === note.id) confirmingId.value = null }, 3000)
}

function snippet(text: string, max = 220) {
  const trimmed = text.replace(/\s+/g, ' ').trim()
  return trimmed.length > max ? trimmed.slice(0, max) + '…' : trimmed
}
</script>

<template>
  <main class="page-shell notes-page">
    <div class="notes-head">
      <PageHeader eyebrow="Apps" title="Notes" description="Save summaries and ideas. Anything you save from a search is linked back to it.">
        <button type="button" class="base-button button-primary" @click="openNew">+ New note</button>
      </PageHeader>
    </div>

    <div v-if="loading" class="notes-loading"><LoadingSpinner /></div>

    <section v-else-if="notes.length" class="notes-grid">
      <article v-for="note in notes" :key="note.id" class="note-card">
        <div class="note-card-head">
          <h2>{{ note.title }}</h2>
          <div class="note-card-actions">
            <button type="button" class="note-action" title="Edit" @click="openEdit(note)">Edit</button>
            <button type="button" class="note-action note-action--danger" :class="{ 'note-action--confirm': confirmingId === note.id }" @click="remove(note)">
              {{ confirmingId === note.id ? 'Confirm?' : 'Delete' }}
            </button>
          </div>
        </div>
        <p class="note-content">{{ snippet(note.content) }}</p>
        <div class="note-card-meta">
          <span v-if="note.sourceQuery" class="note-source">From search: “{{ note.sourceQuery }}”</span>
          <span>{{ relativeTime(note.createdAt) }}</span>
        </div>
      </article>
    </section>

    <EmptyState v-else icon="📝" title="No notes yet" message="Save a search summary, or write your own note — anything you add shows up here.">
      <button type="button" class="base-button button-primary" @click="openNew">Write your first note</button>
    </EmptyState>

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
.notes-page { padding-top: 8px; }
.notes-head :deep(.admin-page-head) { margin-bottom: 22px; }
.notes-loading { display: grid; place-items: center; padding: 60px 0; }
.notes-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px; }
.note-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 20px;
  border: 1px solid var(--color-border);
  border-radius: 16px;
  background: var(--color-surface);
  box-shadow: 0 10px 30px var(--color-shadow);
}
.note-card-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.note-card-head h2 { margin: 0; font-size: 1.02rem; letter-spacing: -.02em; }
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
.note-action--confirm { color: var(--color-danger); background: color-mix(in srgb, var(--color-danger) 12%, var(--color-surface)); }
.note-content {
  margin: 0;
  color: var(--color-muted);
  font-size: .86rem;
  line-height: 1.6;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
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
@media (max-width: 640px) {
  .notes-grid { grid-template-columns: 1fr; }
}
</style>
