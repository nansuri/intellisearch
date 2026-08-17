<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useSiteStore } from '../stores/site'
import FormField from '../components/FormField.vue'
import ThemeToggle from '../components/ThemeToggle.vue'

const auth = useAuthStore()
const site = useSiteStore(); site.load()
const router = useRouter()
const email = ref(''); const password = ref(''); const error = ref('')
const signingIn = ref(false)
const done = ref(false)

async function submit() {
  if (signingIn.value || done.value) return
  error.value = ''
  signingIn.value = true
  try {
    await auth.login(email.value, password.value)
    done.value = true
    // Brief pause so the success animation is visible, then land on the main page.
    window.setTimeout(() => router.push('/'), 650)
  } catch (e) {
    signingIn.value = false
    error.value = (e as Error).message
  }
}

const googleEnabled = computed(() => site.settings?.googleSsoEnabled === true)
function googleSignIn() { window.location.href = '/api/v1/auth/google' }
</script>
<template>
  <main class="login-page">
    <div class="login-page-top"><ThemeToggle /></div>
    <div class="login-card" :class="{ 'login-card--done': done }">
      <div class="login-brand">
        <img v-if="site.settings?.logoUrl" :src="site.settings.logoUrl" class="login-logo" alt="" />
        <span v-else class="brand-mark">{{ (site.settings?.siteName || 'AI').slice(0, 2).toUpperCase() }}</span>
        <div><strong>{{ site.settings?.siteName || 'Intellisearch' }}</strong><span>{{ site.settings?.tagline || 'Research, distilled' }}</span></div>
      </div>
      <h1>Welcome back</h1>
      <p class="login-sub">Sign in to continue asking and keep your profile in sync.</p>

      <div class="login-google" :class="{ 'login-google--done': done }">
        <button v-if="googleEnabled" type="button" class="google-button" :disabled="signingIn || done" @click="googleSignIn">
          <svg class="google-g" viewBox="0 0 24 24" aria-hidden="true"><path fill="#4285F4" d="M23.5 12.27c0-.85-.08-1.66-.22-2.45H12v4.64h6.46a5.53 5.53 0 0 1-2.4 3.63v3.01h3.88c2.27-2.09 3.56-5.17 3.56-8.83Z" /><path fill="#34A853" d="M12 24c3.24 0 5.96-1.07 7.94-2.91l-3.88-3.01c-1.08.72-2.45 1.15-4.06 1.15-3.13 0-5.78-2.11-6.72-4.95H1.27v3.11A12 12 0 0 0 12 24Z" /><path fill="#FBBC05" d="M5.28 14.28a7.2 7.2 0 0 1 0-4.56V6.61H1.27a12 12 0 0 0 0 10.78l4.01-3.11Z" /><path fill="#EA4335" d="M12 4.77c1.76 0 3.34.6 4.58 1.79l3.44-3.44A11.98 11.98 0 0 0 1.27 6.6l4.01 3.12C6.22 6.88 8.87 4.77 12 4.77Z" /></svg>
          <span>{{ signingIn ? 'Signing in…' : 'Continue with Google' }}</span>
        </button>
        <div v-if="googleEnabled" class="login-divider"><span>or continue with email</span></div>
      </div>

      <form class="login-form" @submit.prevent="submit">
        <FormField label="Email"><input v-model="email" type="email" class="text-input" required autocomplete="username" placeholder="you@example.com" /></FormField>
        <FormField label="Password"><input v-model="password" type="password" class="text-input" required autocomplete="current-password" placeholder="••••••••" /></FormField>
        <p v-if="error" class="form-error form-error--block">{{ error }}</p>
        <button type="submit" class="base-button button-primary login-submit" :class="{ 'login-submit--done': done }" :disabled="signingIn || done">
          <span v-if="done" class="login-check" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M5 12.5l4.2 4.2L19 7" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" /></svg></span>
          <span v-else>{{ signingIn ? 'Signing in…' : 'Sign in' }}</span>
        </button>
      </form>
      <RouterLink to="/admin/login" class="login-admin-link">Control panel access</RouterLink>
    </div>
  </main>
</template>

<style scoped>
.login-page { position: relative; display: grid; place-items: center; min-height: 100dvh; padding: 24px; background: var(--color-bg); color: var(--color-text); }
.login-page-top { position: absolute; top: 20px; right: clamp(20px, 4vw, 56px); }
.login-card { width: min(420px, 100%); padding: 36px; border: 1px solid var(--color-border); border-radius: 20px; background: var(--color-surface); box-shadow: 0 30px 60px var(--color-shadow); animation: login-in .55s cubic-bezier(.22, .9, .28, 1) both; }
.login-card > * { animation: login-rise .5s cubic-bezier(.22, .9, .28, 1) both; }
.login-brand { display: flex; align-items: center; gap: 12px; margin-bottom: 26px; animation-delay: .04s; }
.login-brand .brand-mark { width: 42px; height: 42px; border-radius: 12px; font-size: .82rem; background: linear-gradient(135deg, var(--color-primary), color-mix(in srgb, var(--color-primary) 55%, #8b6dff)); color: #fff; }
.login-logo { width: 42px; height: 42px; border-radius: 12px; object-fit: contain; background: var(--color-bg); }
.login-brand div { display: grid; gap: 1px; }
.login-brand strong { font-size: .94rem; letter-spacing: -.02em; }
.login-brand span { color: var(--color-muted); font-size: .74rem; }
.login-card h1 { margin: 0 0 8px; font-size: 1.7rem; letter-spacing: -.04em; animation-delay: .1s; }
.login-sub { margin: 0 0 22px; color: var(--color-muted); font-size: .88rem; line-height: 1.55; animation-delay: .14s; }
.login-google { animation-delay: .18s; }
.google-button { display: flex; align-items: center; justify-content: center; gap: 10px; width: 100%; min-height: 46px; border: 1px solid var(--color-border); border-radius: 12px; background: var(--color-bg); color: var(--color-text); font-size: .88rem; font-weight: 680; cursor: pointer; transition: border-color .16s ease, background .16s ease, transform .16s ease; }
.google-button:hover { border-color: color-mix(in srgb, var(--color-primary) 55%, var(--color-border)); background: var(--color-surface); transform: translateY(-1px); }
.google-button:disabled { opacity: .6; cursor: default; transform: none; }
.google-g { width: 19px; height: 19px; flex: 0 0 auto; }
.login-divider { display: flex; align-items: center; gap: 12px; margin: 20px 0 18px; color: var(--color-muted); font-size: .74rem; }
.login-divider::before, .login-divider::after { content: ''; height: 1px; flex: 1; background: var(--color-border); }
.login-form { display: grid; gap: 16px; animation-delay: .24s; }
.login-submit { width: 100%; margin-top: 10px; min-height: 46px; }
.login-check { display: inline-grid; place-items: center; animation: check-pop .35s cubic-bezier(.22, 1.4, .36, 1) both; }
.login-check svg { width: 20px; height: 20px; }
.login-admin-link { margin-top: 20px; display: block; text-align: center; color: var(--color-muted); font-size: .78rem; font-weight: 650; text-decoration: none; animation-delay: .3s; }
.login-admin-link:hover { color: var(--color-primary); }
.login-card--done { animation: login-out .5s cubic-bezier(.4, 0, .8, 1) forwards; }
@keyframes login-in { from { opacity: 0; transform: translateY(18px) scale(.98); } }
@keyframes login-rise { from { opacity: 0; transform: translateY(10px); } }
@keyframes login-out { to { opacity: 0; transform: translateY(-14px) scale(.98); } }
@keyframes check-pop { from { opacity: 0; transform: scale(.4) rotate(-20deg); } }
@media (prefers-reduced-motion: reduce) { .login-card, .login-card > * { animation: none; } }
</style>
