<script setup lang="ts">
import { onMounted, ref } from 'vue'
import FormField from '../components/FormField.vue'
import Avatar from '../components/Avatar.vue'
import AppHeader from '../components/AppHeader.vue'
import { useSiteStore } from '../stores/site'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'
import { getMe, updateMe, uploadAvatar, type MeResponse } from '../services/api'
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

async function load() {
  loading.value = true
  try { me.value = await getMe(); name.value = me.value.name; email.value = me.value.email } catch { notSignedIn.value = true } finally { loading.value = false }
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
onMounted(() => { if (auth.token) load(); else notSignedIn.value = true })
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
      </template>
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
</style>
