<script setup lang="ts">
import { onMounted, ref } from 'vue'
import FormField from '../components/FormField.vue'
import Avatar from '../components/Avatar.vue'
import AppHeader from '../components/AppHeader.vue'
import { useSiteStore } from '../stores/site'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'
import { getHistory, clearHistory, getMe, updateMe, uploadAvatar, type HistoryItem, type MeResponse } from '../services/api'
import ConfirmModal from '../components/ConfirmModal.vue'
import { useRouter } from 'vue-router'

const site = useSiteStore(); site.load()
const auth = useAuthStore()
const toast = useToastStore()
const router = useRouter()
const me = ref<MeResponse | null>(null)
const loading = ref(true)
const notSignedIn = ref(false)
const saving = ref(false)
const name = ref(''); const email = ref('')
const fileInput = ref<HTMLInputElement | null>(null); const uploading = ref(false)
const history = ref<HistoryItem[]>([])
const historyLoading = ref(false)
const historyError = ref(false)
const clearingHistory = ref(false)
const confirmClear = ref(false)

async function load() {
  loading.value = true
  try { me.value = await getMe(); name.value = me.value.name; email.value = me.value.email } catch { notSignedIn.value = true } finally { loading.value = false }
}
async function loadHistory() {
  historyLoading.value = true
  historyError.value = false
  try { history.value = (await getHistory(50)).items } catch { historyError.value = true } finally { historyLoading.value = false }
}
async function clearAllHistory() {
  clearingHistory.value = true
  try {
    await clearHistory()
    history.value = []
    confirmClear.value = false
    toast.success('Search history cleared.')
  } catch (e) { toast.error((e as Error).message) } finally { clearingHistory.value = false }
}
function timeAgo(iso: string): string {
  const seconds = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 1000))
  if (seconds < 60) return 'just now'
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.round(hours / 24)
  return `${days}d ago`
}
async function saveProfile() {
  if (!name.value.trim() || !email.value.trim()) return
  saving.value = true
  try {
    me.value = { ...me.value!, ...(await updateMe(name.value.trim(), email.value.trim())) }
    auth.user = me.value as unknown as typeof auth.user
    toast.success('Profile saved.')
  } catch (e) { toast.error((e as Error).message) } finally { saving.value = false }
}
async function pickAvatar(file?: File) {
  const f = file || fileInput.value?.files?.[0]
  if (!f || !me.value) return
  uploading.value = true
  try {
    const { avatarUrl } = await uploadAvatar(f)
    me.value.avatarUrl = avatarUrl
    if (auth.user) auth.user = { ...auth.user, avatarUrl }
    toast.success('Avatar updated.')
  } catch (e) { toast.error((e as Error).message) } finally { uploading.value = false; if (fileInput.value) fileInput.value.value = '' }
}
onMounted(() => { if (auth.token) { load(); loadHistory() } else notSignedIn.value = true })
</script>
<template>
  <main class="page-shell account-page">
    <AppHeader />
    <section class="account-content">
      <div v-if="loading" class="admin-loading"><span class="spinner"></span></div>
      <div v-else-if="notSignedIn" class="admin-card admin-card--form">
        <div class="setting-title"><div><div class="section-label">Access</div><h2>Sign in to manage your settings</h2><p>Your profile and avatar appear once you're signed in.</p></div></div>
        <button class="base-button button-primary" style="justify-self:start" @click="router.push('/login')">Go to sign in</button>
      </div>
      <template v-else-if="me">
        <div class="account-heading">
          <div>
            <div class="eyebrow"><span></span> Profile</div>
            <h1>User settings</h1>
            <p>Edit how you appear across the site.</p>
          </div>
          <Avatar :name="me.name" :avatar-url="me.avatarUrl" class="profile-orb" />
        </div>
        <section class="settings-card">
          <div class="setting-title">
            <div><div class="section-label">Identity</div><h2>Your profile</h2><p>Name, email, and avatar.</p></div>
            <span class="status-pill">{{ me.role === 'super_owner' ? 'Super Owner' : 'General user' }}</span>
          </div>
          <div class="profile-preview">
            <button class="avatar-upload" @click="fileInput?.click()">
              <Avatar :name="me.name" :avatar-url="me.avatarUrl" />
              <input ref="fileInput" type="file" accept="image/png,image/jpeg,image/webp,image/gif" hidden @change="pickAvatar()" />
              <span class="avatar-upload-hint">{{ uploading ? 'Uploading…' : 'Change photo' }}</span>
            </button>
            <div class="avatar-upload-copy"><strong>{{ me.name }}</strong><span>{{ me.email }}</span></div>
          </div>
          <div class="admin-form profile-form">
            <FormField label="Full name"><input v-model="name" class="text-input" /></FormField>
            <FormField label="Email"><input v-model="email" type="email" class="text-input" /></FormField>
            <div class="modal-submit-row"><button class="base-button button-primary" :disabled="saving" @click="saveProfile">{{ saving ? 'Saving…' : 'Save profile' }}</button></div>
          </div>
        </section>

        <section class="settings-card">
          <div class="setting-title">
            <div><div class="section-label">History</div><h2>Recent searches</h2><p>Your past questions, used to show recent searches and compose suggestions on the main page.</p></div>
            <button v-if="history.length" class="base-button button-secondary" @click="confirmClear = true">Clear history</button>
          </div>
          <div v-if="historyLoading" class="admin-loading"><span class="spinner"></span></div>
          <p v-else-if="historyError" class="history-note">Couldn't load your search history.</p>
          <p v-else-if="!history.length" class="history-note">No searches yet — your past questions will appear here.</p>
          <ul v-else class="history-list">
            <li v-for="item in history" :key="item.id" class="history-item">
              <span class="history-query" :title="item.query">{{ item.query }}</span>
              <span class="history-time">{{ timeAgo(item.createdAt) }}</span>
            </li>
          </ul>
        </section>
      </template>
      <ConfirmModal v-if="confirmClear" :open="true" title="Clear search history" message="This permanently removes all of your past searches and the suggestions based on them." confirm-label="Clear history" :busy="clearingHistory" @close="confirmClear = false" @confirm="clearAllHistory" />
    </section>
  </main>
</template>

<style scoped>
.avatar-upload { position: relative; display: inline-grid; place-items: center; width: 60px; height: 60px; border: 0; border-radius: 50%; background: transparent; cursor: pointer; }
.avatar-upload .avatar-img, .avatar-upload .avatar-initials { width: 56px; height: 56px; }
.avatar-upload-hint { position: absolute; bottom: -14px; color: var(--color-primary); font-size: .7rem; font-weight: 700; opacity: 0; transition: opacity .15s ease; }
.avatar-upload:hover .avatar-upload-hint { opacity: 1; }
.avatar-upload-copy { display: grid; gap: 2px; }
.avatar-upload-copy strong { font-size: .95rem; }
.avatar-upload-copy span { color: var(--color-muted); font-size: .82rem; }
.profile-form { max-width: 520px; margin-top: 26px; }
.history-note { margin: 18px 0 0; color: var(--color-muted); font-size: .85rem; }
.history-list { display: grid; margin: 22px 0 0; padding: 0; list-style: none; }
.history-item { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; padding: 12px 2px; border-bottom: 1px solid var(--color-border); }
.history-item:last-child { border-bottom: 0; }
.history-query { overflow: hidden; color: var(--color-text); font-size: .9rem; text-overflow: ellipsis; white-space: nowrap; }
.history-time { flex-shrink: 0; color: var(--color-muted); font-size: .74rem; }
@media (max-width: 520px) { .history-item { align-items: flex-start; flex-direction: column; gap: 2px; } }
</style>
