<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useSiteStore } from '../../stores/site'
import Avatar from '../../components/Avatar.vue'
import ThemeToggle from '../../components/ThemeToggle.vue'
import AdminIcon from '../../components/admin/AdminIcon.vue'

const auth = useAuthStore()
const site = useSiteStore(); site.load()
const router = useRouter()
const route = useRoute()

const groups = [
  { label: 'Dashboard', icon: 'dashboard', items: [{ to: '/admin', label: 'Overview' }] },
  { label: 'Users', icon: 'users', items: [{ to: '/admin/users', label: 'All users' }, { to: '/admin/users/suspended', label: 'Suspended' }] },
  { label: 'Statistics', icon: 'stats', items: [{ to: '/admin/stats', label: 'Overview' }, { to: '/admin/stats/top', label: 'Top queries' }, { to: '/admin/stats/usage', label: 'Per-user usage' }, { to: '/admin/stats/ai', label: 'AI service' }] },
  { label: 'AI providers', icon: 'ai', items: [{ to: '/admin/ai/providers', label: 'Providers' }, { to: '/admin/ai/queue', label: 'Queue & limits' }] },
  { label: 'Branding', icon: 'branding', items: [{ to: '/admin/branding/identity', label: 'Identity' }, { to: '/admin/branding/logo', label: 'Logo' }] },
]

const drawerOpen = ref(false)
const collapsed = ref<Record<string, boolean>>({})

function groupActive(label: string) {
  const group = groups.find((g) => g.label === label)
  return !!group?.items.some((item) => item.to === route.path)
}

onMounted(() => {
  for (const g of groups) collapsed.value[g.label] = !groupActive(g.label)
})

watch(() => route.path, () => {
  for (const g of groups) if (groupActive(g.label)) collapsed.value[g.label] = false
})

function toggleGroup(label: string) {
  collapsed.value[label] = !collapsed.value[label]
}

function isActive(item: { to: string }) {
  return route.path === item.to
}

const currentPage = computed(() => {
  for (const g of groups) {
    const item = g.items.find((i) => isActive(i))
    if (item) return item.label
  }
  return 'Overview'
})

function go(item: { to: string }) {
  router.push(item.to)
  drawerOpen.value = false
}

async function logout() { await auth.logout(); router.push('/login') }
</script>
<template>
  <div class="admin-shell">
    <div v-if="drawerOpen" class="admin-drawer-backdrop" @click="drawerOpen = false" />
    <aside class="admin-sidebar" :class="{ 'admin-sidebar--open': drawerOpen }">
      <div class="admin-sidebar-inner">
        <router-link class="admin-sidebar-brand" to="/admin" @click="drawerOpen = false">
          <span class="brand-mark">{{ (site.settings?.siteName || 'CP').slice(0, 2).toUpperCase() }}</span>
          <span class="admin-sidebar-brand-name"><strong>{{ site.settings?.siteName || 'Control Panel' }}</strong><span>Super Owner</span></span>
        </router-link>
        <nav class="admin-nav" aria-label="Control panel">
          <div v-for="g in groups" :key="g.label" class="admin-nav-group" :class="{ 'admin-nav-group--active': groupActive(g.label) }">
            <button type="button" class="admin-nav-group-title" :aria-expanded="!collapsed[g.label]" @click="toggleGroup(g.label)">
              <span class="admin-nav-icon"><AdminIcon :name="g.icon" :size="17" /></span>
              <span class="admin-nav-group-label">{{ g.label }}</span>
              <svg class="admin-chevron" :class="{ 'admin-chevron--open': !collapsed[g.label] }" viewBox="0 0 24 24" aria-hidden="true"><path d="M8.5 10l3.5 3.5L15.5 10" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" /></svg>
            </button>
            <div v-if="!collapsed[g.label]" class="admin-nav-children">
              <button v-for="it in g.items" :key="it.to" type="button" class="admin-nav-link" :class="{ active: isActive(it) }" @click="go(it)">{{ it.label }}</button>
            </div>
          </div>
        </nav>
      </div>
    </aside>
    <div class="admin-main">
      <header class="admin-header">
        <div class="admin-header-title">
          <button class="admin-hamburger" aria-label="Open menu" @click="drawerOpen = true">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h16M4 12h16M4 17h16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" /></svg>
          </button>
          <div class="admin-header-copy"><strong>{{ currentPage }}</strong><span class="admin-header-mobile-label">{{ site.settings?.siteName || 'Control Panel' }}</span></div>
        </div>
        <div class="admin-header-actions">
          <router-link class="admin-site-link admin-site-link--desktop" to="/">View site &#8599;</router-link>
          <ThemeToggle />
          <span class="status-pill">Super Owner</span>
          <span class="admin-avatar"><Avatar :name="auth.user?.name || 'Owner'" :avatar-url="auth.user?.avatarUrl" /></span>
          <button class="base-button button-secondary" @click="logout">Sign out</button>
        </div>
      </header>
      <div class="admin-content">
        <router-view />
      </div>
    </div>
  </div>
</template>
