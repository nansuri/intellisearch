<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useSiteStore } from '../stores/site'
import FormField from '../components/FormField.vue'
import ThemeToggle from '../components/ThemeToggle.vue'

type AuthMode = 'login' | 'register'

const auth = useAuthStore()
const site = useSiteStore(); site.load()
const route = useRoute()
const router = useRouter()

const mode = ref<AuthMode>(route.query.mode === 'register' ? 'register' : 'login')
const name = ref(''); const email = ref(''); const password = ref('')
const error = ref(''); const queryError = ref('')
const busy = ref(false)
const done = ref(false)

const googleEnabled = computed(() => site.settings?.googleSsoEnabled === true)
const title = computed(() => (mode.value === 'register' ? 'Create your account' : 'Welcome back'))
const subtitle = computed(() => (mode.value === 'register' ? 'Start asking questions — it takes less than a minute.' : 'Sign in to keep asking and stay in sync.'))
const submitLabel = computed(() => (mode.value === 'register' ? 'Create account' : 'Sign in'))
const busyLabel = computed(() => (mode.value === 'register' ? 'Creating…' : 'Signing in…'))

function switchMode(next: AuthMode) {
  if (next === mode.value || done.value) return
  mode.value = next
  error.value = ''
  router.replace({ path: '/login', query: next === 'register' ? { mode: 'register' } : {} })
}
watch(() => route.query.mode, (next) => {
  const parsed: AuthMode = next === 'register' ? 'register' : 'login'
  if (parsed !== mode.value) mode.value = parsed
})

async function submit() {
  if (busy.value || done.value) return
  error.value = ''
  busy.value = true
  try {
    if (mode.value === 'register') await auth.register(name.value.trim(), email.value.trim(), password.value)
    else await auth.login(email.value.trim(), password.value)
    done.value = true
    // Brief pause so the success animation is visible, then land on the main page.
    window.setTimeout(() => router.push('/'), 650)
  } catch (e) {
    busy.value = false
    error.value = (e as Error).message
  }
}

function googleSignIn() { window.location.href = '/api/v1/auth/google' }

onMounted(() => {
  const fromQuery = typeof route.query.error === 'string' ? route.query.error : ''
  if (fromQuery) queryError.value = fromQuery
})
</script>
<template>
  <main class="auth-page">
    <div class="auth-blobs" aria-hidden="true">
      <span class="auth-blob auth-blob--a" />
      <span class="auth-blob auth-blob--b" />
      <span class="auth-blob auth-blob--c" />
    </div>
    <div class="auth-top"><ThemeToggle /></div>

    <div class="auth-card" :class="{ 'auth-card--done': done }">
      <div class="auth-brand">
        <img v-if="site.settings?.logoUrl" :src="site.settings.logoUrl" class="auth-logo" alt="" />
        <span v-else class="brand-mark">{{ (site.settings?.siteName || 'AI').slice(0, 2).toUpperCase() }}</span>
        <div><strong>{{ site.settings?.siteName || 'Intellisearch' }}</strong><span>{{ site.settings?.tagline || 'Research, distilled' }}</span></div>
      </div>

      <h1>{{ title }}</h1>
      <p class="auth-sub">{{ subtitle }}</p>

      <p v-if="queryError" class="auth-error auth-error--banner" role="alert">
        {{ queryError }}
        <button type="button" class="auth-error-dismiss" aria-label="Dismiss" @click="queryError = ''">×</button>
      </p>

      <div class="auth-google" :class="{ 'auth-google--done': done }">
        <button v-if="googleEnabled" type="button" class="google-button" :disabled="busy || done" @click="googleSignIn">
          <svg class="google-g" viewBox="0 0 24 24" aria-hidden="true"><path fill="#4285F4" d="M23.5 12.27c0-.85-.08-1.66-.22-2.45H12v4.64h6.46a5.53 5.53 0 0 1-2.4 3.63v3.01h3.88c2.27-2.09 3.56-5.17 3.56-8.83Z" /><path fill="#34A853" d="M12 24c3.24 0 5.96-1.07 7.94-2.91l-3.88-3.01c-1.08.72-2.45 1.15-4.06 1.15-3.13 0-5.78-2.11-6.72-4.95H1.27v3.11A12 12 0 0 0 12 24Z" /><path fill="#FBBC05" d="M5.28 14.28a7.2 7.2 0 0 1 0-4.56V6.61H1.27a12 12 0 0 0 0 10.78l4.01-3.11Z" /><path fill="#EA4335" d="M12 4.77c1.76 0 3.34.6 4.58 1.79l3.44-3.44A11.98 11.98 0 0 0 1.27 6.6l4.01 3.12C6.22 6.88 8.87 4.77 12 4.77Z" /></svg>
          <span>{{ mode === 'register' ? 'Sign up with Google' : 'Continue with Google' }}</span>
        </button>
        <div v-if="googleEnabled" class="auth-divider"><span>or {{ mode === 'register' ? 'create an account' : 'continue' }} with email</span></div>
      </div>

      <div class="auth-tabs" role="tablist" aria-label="Sign in or register">
        <button type="button" role="tab" :aria-selected="mode === 'login'" :class="{ active: mode === 'login' }" @click="switchMode('login')">Sign in</button>
        <button type="button" role="tab" :aria-selected="mode === 'register'" :class="{ active: mode === 'register' }" @click="switchMode('register')">Create account</button>
      </div>

      <Transition name="auth-form" mode="out-in">
        <form :key="mode" class="auth-form" @submit.prevent="submit">
          <FormField v-if="mode === 'register'" label="Full name">
            <input v-model="name" class="auth-input" type="text" required autocomplete="name" placeholder="Jane Doe" />
          </FormField>
          <FormField label="Email">
            <input v-model="email" class="auth-input" type="email" required autocomplete="username" placeholder="you@example.com" />
          </FormField>
          <FormField :label="mode === 'register' ? 'Password' : 'Password'" :hint="mode === 'register' ? 'At least 8 characters' : undefined">
            <input v-model="password" class="auth-input" type="password" required minlength="8" :autocomplete="mode === 'register' ? 'new-password' : 'current-password'" placeholder="••••••••" />
          </FormField>
          <p v-if="error" class="auth-error" role="alert">{{ error }}</p>
          <button type="submit" class="base-button auth-submit" :class="{ 'auth-submit--done': done }" :disabled="busy || done">
            <span v-if="done" class="auth-check" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M5 12.5l4.2 4.2L19 7" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" /></svg></span>
            <span v-else>{{ busy ? busyLabel : submitLabel }}</span>
          </button>
        </form>
      </Transition>
    </div>
  </main>
</template>

<style scoped>
.auth-page { position: relative; display: grid; place-items: center; min-height: 100dvh; padding: 24px; overflow: hidden; background: var(--color-bg); color: var(--color-text); }
.auth-top { position: absolute; top: 20px; right: clamp(20px, 4vw, 56px); z-index: 3; }

/* Aurora background blobs */
.auth-blobs { position: absolute; inset: 0; z-index: 0; pointer-events: none; }
.auth-blob { position: absolute; width: 46vmax; height: 46vmax; border-radius: 50%; filter: blur(90px); opacity: .5; }
.auth-blob--a { top: -18vmax; left: -10vmax; background: radial-gradient(circle at 30% 30%, color-mix(in srgb, var(--color-primary) 55%, transparent), transparent 70%); animation: auth-float-a 26s ease-in-out infinite alternate; }
.auth-blob--b { bottom: -20vmax; right: -12vmax; background: radial-gradient(circle at 60% 40%, color-mix(in srgb, #8b6dff 45%, transparent), transparent 70%); animation: auth-float-b 32s ease-in-out infinite alternate; }
.auth-blob--c { top: 38%; left: 55%; width: 34vmax; height: 34vmax; background: radial-gradient(circle at 50% 50%, color-mix(in srgb, var(--color-focus) 38%, transparent), transparent 70%); animation: auth-float-c 22s ease-in-out infinite alternate; }

/* Glass card */
.auth-card { position: relative; z-index: 2; width: min(430px, 100%); padding: 38px 36px 32px; border: 1px solid color-mix(in srgb, var(--color-border) 65%, transparent); border-radius: 26px; background: color-mix(in srgb, var(--color-surface) 62%, transparent); box-shadow: 0 30px 70px var(--color-shadow); backdrop-filter: blur(24px) saturate(1.45); -webkit-backdrop-filter: blur(24px) saturate(1.45); animation: auth-in .6s cubic-bezier(.22, .9, .28, 1) both; }
.auth-card::before { content: ''; position: absolute; inset: 0; border-radius: 26px; background: linear-gradient(135deg, color-mix(in srgb, #fff 9%, transparent), transparent 40%, color-mix(in srgb, #fff 5%, transparent)); pointer-events: none; }
.auth-card > * { position: relative; }
.auth-card > *:not(.auth-tabs) { animation: auth-rise .55s cubic-bezier(.22, .9, .28, 1) both; }

.auth-brand { display: flex; align-items: center; gap: 12px; margin-bottom: 24px; animation-delay: .04s; }
.auth-brand .brand-mark { width: 42px; height: 42px; border-radius: 13px; font-size: .82rem; background: linear-gradient(135deg, var(--color-primary), color-mix(in srgb, var(--color-primary) 45%, #8b6dff)); color: #fff; box-shadow: 0 8px 22px color-mix(in srgb, var(--color-primary) 35%, transparent); }
.auth-logo { width: 42px; height: 42px; border-radius: 13px; object-fit: contain; background: color-mix(in srgb, var(--color-bg) 70%, transparent); }
.auth-brand div { display: grid; gap: 1px; }
.auth-brand strong { font-size: .94rem; letter-spacing: -.02em; }
.auth-brand span { color: var(--color-muted); font-size: .74rem; }
.auth-card h1 { margin: 0 0 8px; font-size: 1.72rem; letter-spacing: -.04em; animation-delay: .1s; }
.auth-sub { margin: 0 0 22px; color: var(--color-muted); font-size: .88rem; line-height: 1.55; animation-delay: .14s; }

.auth-error { margin: 0 0 14px; padding: 10px 12px; border: 1px solid color-mix(in srgb, var(--color-danger) 35%, var(--color-border)); border-radius: 10px; background: color-mix(in srgb, var(--color-danger) 10%, transparent); color: var(--color-danger); font-size: .8rem; line-height: 1.45; }
.auth-error--banner { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.auth-error-dismiss { border: 0; background: transparent; color: inherit; cursor: pointer; font-size: 1rem; line-height: 1; }

.auth-google { animation-delay: .18s; }
.google-button { display: flex; align-items: center; justify-content: center; gap: 10px; width: 100%; min-height: 48px; border: 1px solid color-mix(in srgb, var(--color-border) 75%, transparent); border-radius: 13px; background: color-mix(in srgb, var(--color-bg) 45%, transparent); color: var(--color-text); font-size: .88rem; font-weight: 680; cursor: pointer; backdrop-filter: blur(8px); -webkit-backdrop-filter: blur(8px); transition: border-color .16s ease, background .16s ease, transform .16s ease, box-shadow .16s ease; }
.google-button:hover { border-color: color-mix(in srgb, var(--color-primary) 55%, var(--color-border)); background: color-mix(in srgb, var(--color-bg) 70%, transparent); transform: translateY(-1px); box-shadow: 0 8px 20px var(--color-shadow); }
.google-button:disabled { opacity: .6; cursor: default; transform: none; }
.google-g { width: 19px; height: 19px; flex: 0 0 auto; }
.auth-divider { display: flex; align-items: center; gap: 12px; margin: 20px 0 18px; color: var(--color-muted); font-size: .74rem; }
.auth-divider::before, .auth-divider::after { content: ''; height: 1px; flex: 1; background: color-mix(in srgb, var(--color-border) 70%, transparent); }

/* Segmented tabs */
.auth-tabs { display: grid; grid-template-columns: 1fr 1fr; gap: 4px; margin-bottom: 20px; padding: 4px; border: 1px solid color-mix(in srgb, var(--color-border) 55%, transparent); border-radius: 13px; background: color-mix(in srgb, var(--color-bg) 38%, transparent); }
.auth-tabs button { min-height: 38px; border: 0; border-radius: 10px; background: transparent; color: var(--color-muted); font-size: .82rem; font-weight: 700; cursor: pointer; transition: color .18s ease, background .18s ease, box-shadow .18s ease; }
.auth-tabs button.active { color: var(--color-text); background: color-mix(in srgb, var(--color-surface) 88%, transparent); box-shadow: 0 4px 14px var(--color-shadow); }

.auth-form { display: grid; gap: 15px; }
.auth-form :deep(.form-label) { color: var(--color-muted); font-size: .72rem; font-weight: 720; letter-spacing: .06em; text-transform: uppercase; }
.auth-input { width: 100%; min-height: 46px; padding: 0 14px; border: 1px solid color-mix(in srgb, var(--color-border) 70%, transparent); border-radius: 12px; background: color-mix(in srgb, var(--color-bg) 42%, transparent); color: var(--color-text); font-size: .88rem; backdrop-filter: blur(8px); -webkit-backdrop-filter: blur(8px); transition: border-color .15s ease, box-shadow .15s ease, background .15s ease; }
.auth-input:focus { border-color: var(--color-primary); outline: 0; box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-focus) 30%, transparent); background: color-mix(in srgb, var(--color-bg) 60%, transparent); }
.auth-input::placeholder { color: var(--color-muted); }

.auth-submit { width: 100%; margin-top: 6px; min-height: 48px; border: 0; border-radius: 13px; background: linear-gradient(135deg, var(--color-primary), color-mix(in srgb, var(--color-primary) 60%, #8b6dff)); color: #fff; font-size: .9rem; font-weight: 720; box-shadow: 0 12px 28px color-mix(in srgb, var(--color-primary) 38%, transparent); transition: transform .16s ease, box-shadow .16s ease, filter .16s ease; }
.auth-submit:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 16px 34px color-mix(in srgb, var(--color-primary) 45%, transparent); filter: brightness(1.04); }
.auth-submit:disabled { opacity: .8; cursor: default; }
.auth-check { display: inline-grid; place-items: center; animation: check-pop .35s cubic-bezier(.22, 1.4, .36, 1) both; }
.auth-check svg { width: 20px; height: 20px; }

.auth-card--done { animation: auth-out .5s cubic-bezier(.4, 0, .8, 1) forwards; }

/* Form switch */
.auth-form-enter-active, .auth-form-leave-active { transition: opacity .2s ease, transform .2s ease; }
.auth-form-enter-from { opacity: 0; transform: translateY(8px); }
.auth-form-leave-to { opacity: 0; transform: translateY(-8px); }

@keyframes auth-float-a { from { transform: translate(0, 0) scale(1); } to { transform: translate(6vmax, 4vmax) scale(1.12); } }
@keyframes auth-float-b { from { transform: translate(0, 0) scale(1.05); } to { transform: translate(-5vmax, -5vmax) scale(.95); } }
@keyframes auth-float-c { from { transform: translate(0, 0) scale(1); } to { transform: translate(-4vmax, 3vmax) scale(1.18); } }
@keyframes auth-in { from { opacity: 0; transform: translateY(20px) scale(.97); } }
@keyframes auth-rise { from { opacity: 0; transform: translateY(10px); } }
@keyframes auth-out { to { opacity: 0; transform: translateY(-14px) scale(.98); } }
@keyframes check-pop { from { opacity: 0; transform: scale(.4) rotate(-20deg); } }
@media (prefers-reduced-motion: reduce) { .auth-blob, .auth-card, .auth-card > *, .auth-check { animation: none; } }
</style>
