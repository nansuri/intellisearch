<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AdminIcon from './admin/AdminIcon.vue'

const router = useRouter()
const open = ref(false)
const root = ref<HTMLElement | null>(null)

const apps = [
  { to: '/apps', label: 'Mini Apps Studio', desc: 'Build and publish mini apps', icon: 'dashboard' },
  { to: '/notes', label: 'Notes', desc: 'Save and organize summaries', icon: 'note' },
  { to: '/translator', label: 'Translator', desc: 'Google-style translate', icon: 'translate' },
]

function toggle() { open.value = !open.value }
function close() { open.value = false }
function go(to: string) { close(); router.push(to) }

function onDocClick(e: MouseEvent) {
  if (!open.value || !root.value) return
  if (!root.value.contains(e.target as Node)) close()
}
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') close()
}

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
  <div ref="root" class="app-drawer">
    <button
      type="button"
      class="app-drawer-trigger"
      :class="{ 'app-drawer-trigger--open': open }"
      :aria-expanded="open"
      aria-haspopup="true"
      aria-label="Open apps"
      title="Apps"
      @click.stop="toggle"
    >
      <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true">
        <rect x="3" y="3" width="7" height="7" rx="1.5" />
        <rect x="14" y="3" width="7" height="7" rx="1.5" />
        <rect x="3" y="14" width="7" height="7" rx="1.5" />
        <rect x="14" y="14" width="7" height="7" rx="1.5" />
      </svg>
    </button>

    <Transition name="app-drawer-fade">
      <div v-if="open" class="app-drawer-panel" role="menu" @click.stop>
        <div class="app-drawer-label">Apps</div>
        <div class="app-drawer-grid">
          <button v-for="app in apps" :key="app.to" type="button" class="app-drawer-app" role="menuitem" @click="go(app.to)">
            <span class="app-drawer-icon" aria-hidden="true"><AdminIcon :name="app.icon" :size="20" /></span>
            <span class="app-drawer-app-copy">
              <strong>{{ app.label }}</strong>
              <span>{{ app.desc }}</span>
            </span>
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.app-drawer { position: relative; }
.app-drawer-trigger {
  display: inline-grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border: 1px solid var(--color-border);
  border-radius: 50%;
  background: var(--color-surface);
  color: var(--color-muted);
  cursor: pointer;
  transition: border-color .16s ease, color .16s ease, transform .16s ease;
}
.app-drawer-trigger:hover { border-color: var(--color-text); color: var(--color-text); transform: translateY(-1px); }
.app-drawer-trigger--open { border-color: var(--color-primary); color: var(--color-primary); }
.app-drawer-panel {
  position: absolute;
  top: calc(100% + 10px);
  right: 0;
  z-index: 50;
  width: min(300px, calc(100vw - 40px));
  padding: 16px;
  border: 1px solid var(--color-border);
  border-radius: 16px;
  background: var(--color-surface);
  box-shadow: 0 16px 48px var(--color-shadow);
}
.app-drawer-label {
  margin-bottom: 10px;
  color: var(--color-muted);
  font-size: .68rem;
  font-weight: 720;
  letter-spacing: .08em;
  text-transform: uppercase;
}
.app-drawer-grid { display: grid; gap: 6px; }
.app-drawer-app {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 10px 12px;
  border: 0;
  border-radius: 12px;
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  text-align: left;
  transition: background .14s ease;
}
.app-drawer-app:hover { background: var(--color-surface-subtle); }
.app-drawer-icon {
  display: inline-grid;
  place-items: center;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  border-radius: 11px;
  background: linear-gradient(135deg, var(--color-surface-subtle), color-mix(in srgb, var(--color-primary) 16%, var(--color-surface-subtle)));
  color: var(--color-primary);
}
.app-drawer-app-copy { display: grid; gap: 1px; min-width: 0; }
.app-drawer-app-copy strong { font-size: .88rem; letter-spacing: -.01em; }
.app-drawer-app-copy span { color: var(--color-muted); font-size: .76rem; }
.app-drawer-fade-enter-active,
.app-drawer-fade-leave-active { transition: opacity .15s ease, transform .15s ease; }
.app-drawer-fade-enter-from,
.app-drawer-fade-leave-to { opacity: 0; transform: translateY(-6px); }
</style>
