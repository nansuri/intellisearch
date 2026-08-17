<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useSiteStore } from '../stores/site'
import FormField from '../components/FormField.vue'
import ThemeToggle from '../components/ThemeToggle.vue'

const auth = useAuthStore()
const site = useSiteStore(); site.load()
const router = useRouter()
const email = ref(''); const password = ref(''); const error = ref('')

async function submit() {
  error.value = ''
  try {
    const user = await auth.login(email.value, password.value)
    router.push(user.role === 'super_owner' ? '/admin' : '/account')
  } catch (e) { error.value = (e as Error).message }
}
</script>
<template>
  <main class="login-page">
    <div class="login-page-top"><ThemeToggle /></div>
    <div class="login-card">
      <div class="login-brand">
        <img v-if="site.settings?.logoUrl" :src="site.settings.logoUrl" class="login-logo" alt="" />
        <span v-else class="brand-mark">{{ (site.settings?.siteName || 'AI').slice(0, 2).toUpperCase() }}</span>
        <div><strong>{{ site.settings?.siteName || 'Intellisearch' }}</strong><span>{{ site.settings?.tagline || 'Research, distilled' }}</span></div>
      </div>
      <h1>Sign in</h1>
      <p class="login-sub">Sign in to manage your profile, avatar, and AI usage.</p>
      <form class="login-form" @submit.prevent="submit">
        <FormField label="Email"><input v-model="email" type="email" class="text-input" required autocomplete="username" placeholder="you@example.com" /></FormField>
        <FormField label="Password"><input v-model="password" type="password" class="text-input" required autocomplete="current-password" placeholder="••••••••" /></FormField>
        <p v-if="error" class="form-error form-error--block">{{ error }}</p>
        <button type="submit" class="base-button button-primary login-submit" :disabled="auth.loading">{{ auth.loading ? 'Signing in…' : 'Sign in' }}</button>
      </form>
      <RouterLink to="/admin/login" class="login-admin-link">Control panel access</RouterLink>
    </div>
  </main>
</template>

<style scoped>
.login-page { position: relative; display: grid; place-items: center; min-height: 100dvh; padding: 24px; background: var(--color-bg); color: var(--color-text); }
.login-page-top { position: absolute; top: 20px; right: clamp(20px, 4vw, 56px); }
.login-card { width: min(420px, 100%); padding: 36px; border: 1px solid var(--color-border); border-radius: 20px; background: var(--color-surface); box-shadow: 0 30px 60px var(--color-shadow); }
.login-brand { display: flex; align-items: center; gap: 12px; margin-bottom: 26px; }
.login-brand .brand-mark { width: 42px; height: 42px; border-radius: 12px; font-size: .82rem; background: linear-gradient(135deg, var(--color-primary), color-mix(in srgb, var(--color-primary) 55%, #8b6dff)); color: #fff; }
.login-logo { width: 42px; height: 42px; border-radius: 12px; object-fit: contain; background: var(--color-bg); }
.login-brand div { display: grid; gap: 1px; }
.login-brand strong { font-size: .94rem; letter-spacing: -.02em; }
.login-brand span { color: var(--color-muted); font-size: .74rem; }
.login-card h1 { margin: 0 0 8px; font-size: 1.7rem; letter-spacing: -.04em; }
.login-sub { margin: 0 0 22px; color: var(--color-muted); font-size: .88rem; line-height: 1.55; }
.login-form { display: grid; gap: 16px; }
.login-submit { width: 100%; margin-top: 10px; }
.login-admin-link { margin-top: 20px; display: block; text-align: center; color: var(--color-muted); font-size: .78rem; font-weight: 650; text-decoration: none; }
.login-admin-link:hover { color: var(--color-primary); }
</style>
