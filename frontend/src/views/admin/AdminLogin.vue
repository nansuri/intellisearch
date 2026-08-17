<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useSiteStore } from '../../stores/site'
import FormField from '../../components/FormField.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import ThemeToggle from '../../components/ThemeToggle.vue'
import { useToastStore } from '../../stores/toast'

const auth = useAuthStore()
const site = useSiteStore(); site.load()
const toast = useToastStore()
const router = useRouter()
const email = ref(''); const password = ref(''); const error = ref('')

async function submit() {
  error.value = ''
  try {
    const user = await auth.login(email.value, password.value)
    if (user.role !== 'super_owner') { router.push('/account'); return }
    toast.success('Welcome back.')
    router.push('/admin')
  } catch (e) { error.value = (e as Error).message }
}
</script>
<template>
  <main class="admin-login">
    <div class="admin-login-top"><ThemeToggle /></div>
    <div class="admin-login-card">
      <div class="admin-login-brand"><img v-if="site.settings?.logoUrl" :src="site.settings.logoUrl" class="admin-login-logo" alt="" /><span class="brand-mark">{{ (site.settings?.siteName || 'CP').slice(0, 2).toUpperCase() }}</span><div><strong>{{ site.settings?.siteName || 'Control Panel' }}</strong><span>Super Owner access</span></div></div>
      <h1>Sign in</h1>
      <p class="admin-login-sub">Use your Super Owner credentials to manage users, AI providers, and branding.</p>
      <form class="admin-form" @submit.prevent="submit">
        <FormField label="Email"><input v-model="email" type="email" class="text-input" required autocomplete="username" placeholder="owner@example.com" /></FormField>
        <FormField label="Password"><input v-model="password" type="password" class="text-input" required autocomplete="current-password" placeholder="••••••••" /></FormField>
        <p v-if="error" class="form-error form-error--block">{{ error }}</p>
        <button type="submit" class="base-button button-primary admin-login-submit" :disabled="auth.loading">{{ auth.loading ? 'Signing in…' : 'Sign in' }}</button>
      </form>
      <LoadingSpinner v-if="auth.loading" />
      <RouterLink to="/login" class="admin-login-user-link">Not a Super Owner? Sign in to your account</RouterLink>
    </div>
  </main>
</template>