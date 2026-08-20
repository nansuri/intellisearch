<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import Avatar from './Avatar.vue'
import { useAuthStore } from '../stores/auth'
import { getMe, type MeResponse } from '../services/api'

const auth = useAuthStore()
const router = useRouter()
const open = ref(false)
const root = ref<HTMLElement | null>(null)
const me = ref<MeResponse | null>(null)
const loading = ref(false)

const usedPct = computed(() => {
  if (!me.value) return 0
  const { quota, usedToday } = me.value.usage
  return quota > 0 ? Math.min(100, Math.round((usedToday / quota) * 100)) : 0
})

async function loadUsage() {
  if (!auth.isAuthed) { me.value = null; return }
  loading.value = true
  try { me.value = await getMe() } catch { me.value = null } finally { loading.value = false }
}

function toggle() {
  open.value = !open.value
  if (open.value) loadUsage()
}

function close() { open.value = false }

function onDocClick(e: MouseEvent) {
  if (!open.value || !root.value) return
  if (!root.value.contains(e.target as Node)) close()
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') close()
}

async function logout() {
  close()
  await auth.logout()
  router.push('/')
}

function goSettings() { close(); router.push('/account') }
function goHistory() { close(); router.push('/history') }
function goStudio() { close(); router.push('/apps') }
function goSignIn() { close(); router.push('/login') }
function goControlPanel() { close(); router.push('/admin') }

watch(() => auth.isAuthed, () => { if (!auth.isAuthed) me.value = null })

onMounted(() => {
  document.addEventListener('click', onDocClick, true)
  document.addEventListener('keydown', onKeydown)
})
onUnmounted(() => {
  document.removeEventListener('click', onDocClick, true)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div ref="root" class="account-menu">
    <button
      type="button"
      class="avatar account-menu-trigger"
      :class="{ 'account-menu-trigger--open': open }"
      :aria-expanded="open"
      aria-haspopup="true"
      aria-label="Open account menu"
      :title="auth.user?.name || 'Account'"
      @click.stop="toggle"
    >
      <Avatar v-if="auth.user" :name="auth.user.name" :avatar-url="auth.user.avatarUrl" />
      <span v-else>?</span>
    </button>

    <Transition name="account-menu-fade">
      <div v-if="open" class="account-menu-panel" role="menu" @click.stop>
        <template v-if="auth.isAuthed && auth.user">
          <div class="account-menu-profile">
            <Avatar :name="auth.user.name" :avatar-url="auth.user.avatarUrl" class="account-menu-avatar" />
            <div class="account-menu-identity">
              <strong>{{ auth.user.name }}</strong>
              <span>{{ auth.user.email }}</span>
            </div>
          </div>

          <div class="account-menu-section">
            <div class="account-menu-section-label">AI limit &amp; usage</div>
            <div v-if="loading" class="account-menu-skeleton" />
            <template v-else-if="me">
              <div class="account-menu-usage-track">
                <div class="account-menu-usage-fill" :style="{ width: `${usedPct}%` }" />
              </div>
              <div class="account-menu-usage-caption">
                <span>{{ me.usage.usedToday }} used</span>
                <span>{{ me.usage.quota > 0 ? `${me.usage.remaining} left today` : 'Unlimited' }}</span>
              </div>
            </template>
          </div>

          <div class="account-menu-section account-menu-actions">
            <div class="account-menu-section-label">Session</div>
            <button type="button" class="account-menu-item" role="menuitem" @click="goSettings">User settings</button>
            <button type="button" class="account-menu-item" role="menuitem" @click="goHistory">Search history</button>
            <button type="button" class="account-menu-item" role="menuitem" @click="goStudio">Mini Apps Studio</button>
            <button v-if="auth.isSuperOwner" type="button" class="account-menu-item" role="menuitem" @click="goControlPanel">Control panel</button>
            <button type="button" class="account-menu-item account-menu-item--danger" role="menuitem" @click="logout">Log out</button>
          </div>
        </template>

        <template v-else>
          <div class="account-menu-guest">
            <p>Sign in to see your profile, AI usage, and session.</p>
            <button type="button" class="base-button button-primary account-menu-signin" @click="goSignIn">Sign in</button>
          </div>
        </template>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.account-menu { position: relative; }
.account-menu-trigger { cursor: pointer; border: 0; font: inherit; }
.account-menu-trigger--open { border-color: var(--color-primary); }
.account-menu-trigger .avatar-img,
.account-menu-trigger .avatar-initials { width: 34px; height: 34px; }
.account-menu-panel {
  position: absolute;
  top: calc(100% + 10px);
  right: 0;
  z-index: 50;
  width: min(320px, calc(100vw - 40px));
  padding: 16px;
  border: 1px solid var(--color-border);
  border-radius: 16px;
  background: var(--color-surface);
  box-shadow: 0 16px 48px var(--color-shadow);
}
.account-menu-profile {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--color-border);
}
.account-menu-avatar :deep(.avatar-img),
.account-menu-avatar :deep(.avatar-initials) { width: 44px; height: 44px; }
.account-menu-identity { display: grid; gap: 2px; min-width: 0; }
.account-menu-identity strong { font-size: .92rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.account-menu-identity span { color: var(--color-muted); font-size: .78rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.account-menu-section { padding-top: 14px; }
.account-menu-section-label {
  margin-bottom: 10px;
  color: var(--color-muted);
  font-size: .68rem;
  font-weight: 720;
  letter-spacing: .08em;
  text-transform: uppercase;
}
.account-menu-usage-track {
  height: 8px;
  border-radius: 999px;
  background: var(--color-surface-subtle);
  overflow: hidden;
}
.account-menu-usage-fill {
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, var(--color-primary), color-mix(in srgb, var(--color-primary) 55%, var(--color-focus)));
  transition: width .35s ease;
}
.account-menu-usage-caption {
  display: flex;
  justify-content: space-between;
  margin-top: 6px;
  color: var(--color-muted);
  font-size: .74rem;
}
.account-menu-skeleton {
  height: 8px;
  border-radius: 999px;
  background: var(--color-surface-subtle);
  animation: skeleton-pulse 1.4s ease-in-out infinite;
}
.account-menu-actions { display: grid; gap: 4px; }
.account-menu-item {
  display: block;
  width: 100%;
  padding: 10px 12px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  font-size: .84rem;
  font-weight: 620;
  text-align: left;
  transition: background .14s ease;
}
.account-menu-item:hover { background: var(--color-surface-subtle); }
.account-menu-item--danger { color: var(--color-danger); }
.account-menu-guest p { margin: 0 0 14px; color: var(--color-muted); font-size: .84rem; line-height: 1.5; }
.account-menu-signin { width: 100%; }
.account-menu-fade-enter-active,
.account-menu-fade-leave-active { transition: opacity .15s ease, transform .15s ease; }
.account-menu-fade-enter-from,
.account-menu-fade-leave-to { opacity: 0; transform: translateY(-6px); }
</style>
