<script setup lang="ts">
import ThemeToggle from './ThemeToggle.vue'
import AccountMenu from './AccountMenu.vue'
import AppDrawer from './AppDrawer.vue'
import { useSiteStore } from '../stores/site'
import { useAuthStore } from '../stores/auth'
import { useRouter } from 'vue-router'

withDefaults(defineProps<{ compact?: boolean; sticky?: boolean }>(), { compact: false, sticky: true })
const site = useSiteStore()
const auth = useAuthStore()
const router = useRouter()
function adminGo() { router.push('/admin/users') }
</script>

<template>
  <div class="header-bar" :class="{ 'header-bar--sticky': sticky }">
    <header class="app-header" :class="{ 'app-header--compact': compact }">
      <RouterLink to="/" class="brand-link" aria-label="Go to home">
        <img v-if="site.settings?.logoUrl" class="brand-logo" :src="site.settings.logoUrl" alt="" />
        <span v-else class="brand-mark" aria-hidden="true">{{ (site.settings?.siteName || 'A').slice(0, 2).toUpperCase() }}</span>
        <span class="brand-copy">
          <span class="site-name">{{ site.settings?.siteName || 'Intellisearch' }}</span>
          <span class="brand-caption">{{ site.settings?.tagline || 'Research, distilled' }}</span>
        </span>
      </RouterLink>
      <div v-if="$slots.center" class="header-center"><slot name="center" /></div>
      <nav class="header-actions" aria-label="Account actions">
        <ThemeToggle />
        <button v-if="auth.isSuperOwner" class="admin-entry" @click="adminGo">Control panel</button>
        <AppDrawer v-if="auth.isAuthed" />
        <AccountMenu />
      </nav>
    </header>
  </div>
</template>

<style scoped>
.admin-entry { min-height: 36px; padding: 0 14px; border: 1px solid var(--color-border); border-radius: 10px; background: var(--color-surface); color: var(--color-text); font-size: .78rem; font-weight: 680; cursor: pointer; transition: border-color .16s ease; }
.admin-entry:hover { border-color: var(--color-primary); color: var(--color-primary); }
</style>
