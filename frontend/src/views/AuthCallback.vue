<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useSiteStore } from '../stores/site'
import ThemeToggle from '../components/ThemeToggle.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const site = useSiteStore(); site.load()
const status = ref<'loading' | 'error'>('loading')
const message = ref('Completing sign-in…')

onMounted(async () => {
  const token = String(route.query.token || '')
  if (!token) {
    status.value = 'error'
    message.value = 'Sign-in link is missing or invalid.'
    return
  }
  try {
    await auth.setSession(token)
    // Brief pause so the success animation is visible before landing home.
    message.value = 'Welcome!'
    window.setTimeout(() => router.replace('/'), 900)
  } catch {
    status.value = 'error'
    message.value = 'Sign-in could not be completed. Please try again.'
  }
})
</script>

<template>
  <main class="auth-callback-page">
    <div class="auth-callback-top"><ThemeToggle /></div>
    <div class="auth-callback-card">
      <div class="auth-callback-mark" :class="{ 'auth-callback-mark--error': status === 'error' }">
        <svg v-if="status === 'loading'" class="auth-callback-spinner" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-dasharray="46 20" /></svg>
        <svg v-else-if="status === 'error'" class="auth-callback-badge" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 21a9 9 0 1 1 0-18 9 9 0 0 1 0 18Zm-1.2-6.2 4.5-4.5-1.2-1.2-3.3 3.3-1.4-1.4-1.2 1.2 2.6 2.6Z" fill="currentColor" /></svg>
        <svg v-else class="auth-callback-badge" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 21a9 9 0 1 1 0-18 9 9 0 0 1 0 18Zm4.2-11.2-1.2-1.2-3.8 3.8-1.6-1.6-1.2 1.2 2.8 2.8 5-5Z" fill="currentColor" /></svg>
      </div>
      <div class="auth-callback-brand">
        <span class="brand-mark">{{ (site.settings?.siteName || 'AI').slice(0, 2).toUpperCase() }}</span>
        <div><strong>{{ site.settings?.siteName || 'Intellisearch' }}</strong><span>{{ site.settings?.tagline || 'Research, distilled' }}</span></div>
      </div>
      <p class="auth-callback-message" :class="{ 'auth-callback-message--error': status === 'error' }">{{ message }}</p>
      <RouterLink v-if="status === 'error'" to="/login" class="auth-callback-back">Back to sign in</RouterLink>
    </div>
  </main>
</template>

<style scoped>
.auth-callback-page { position: relative; display: grid; place-items: center; min-height: 100dvh; padding: 24px; background: var(--color-bg); color: var(--color-text); }
.auth-callback-top { position: absolute; top: 20px; right: clamp(20px, 4vw, 56px); }
.auth-callback-card { display: grid; justify-items: center; gap: 18px; width: min(380px, 100%); padding: 44px 32px; border: 1px solid var(--color-border); border-radius: 22px; background: var(--color-surface); box-shadow: 0 30px 60px var(--color-shadow); text-align: center; animation: callback-in .5s cubic-bezier(.22, .9, .28, 1) both; }
.auth-callback-mark { display: grid; place-items: center; width: 72px; height: 72px; border-radius: 50%; color: var(--color-primary); background: color-mix(in srgb, var(--color-primary) 12%, var(--color-surface)); }
.auth-callback-mark--error { color: var(--color-danger); background: color-mix(in srgb, var(--color-danger) 12%, var(--color-surface)); }
.auth-callback-spinner { width: 34px; height: 34px; animation: callback-spin 1s linear infinite; }
.auth-callback-badge { width: 34px; height: 34px; animation: callback-pop .35s cubic-bezier(.22, 1.4, .36, 1) both; }
.auth-callback-brand { display: flex; align-items: center; gap: 12px; }
.auth-callback-brand .brand-mark { width: 40px; height: 40px; border-radius: 12px; font-size: .8rem; background: linear-gradient(135deg, var(--color-primary), color-mix(in srgb, var(--color-primary) 55%, #8b6dff)); color: #fff; }
.auth-callback-brand div { display: grid; gap: 1px; text-align: left; }
.auth-callback-brand strong { font-size: .94rem; letter-spacing: -.02em; }
.auth-callback-brand span:not(.brand-mark) { color: var(--color-muted); font-size: .74rem; }
.auth-callback-message { margin: 0; color: var(--color-muted); font-size: .9rem; }
.auth-callback-message--error { color: var(--color-danger); }
.auth-callback-back { margin-top: 4px; color: var(--color-primary); font-size: .8rem; font-weight: 650; text-decoration: none; }
.auth-callback-back:hover { text-decoration: underline; }
@keyframes callback-in { from { opacity: 0; transform: translateY(14px) scale(.98); } }
@keyframes callback-spin { to { transform: rotate(360deg); } }
@keyframes callback-pop { from { opacity: 0; transform: scale(.4); } }
</style>
