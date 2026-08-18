<script setup lang="ts">
import { computed } from 'vue'

/** SF-style stroke icons for the control panel (Apple-ish design language). */
const props = withDefaults(defineProps<{ name: string; size?: number }>(), { size: 18 })

const paths = computed(() => ICONS[props.name] || ICONS.dashboard)

const ICONS: Record<string, string> = {
  dashboard: '<rect x="3" y="3" width="8" height="8" rx="2"/><rect x="13" y="3" width="8" height="8" rx="2"/><rect x="3" y="13" width="8" height="8" rx="2"/><rect x="13" y="13" width="8" height="8" rx="2"/>',
  users: '<path d="M16 19v-1.5a3.5 3.5 0 0 0-3.5-3.5h-5A3.5 3.5 0 0 0 4 17.5V19"/><circle cx="9.5" cy="8.5" r="3"/><path d="M17.5 19v-1.2a2.8 2.8 0 0 0-2-2.7"/><path d="M16.5 8.8a2.8 2.8 0 0 1 0 5.4"/>',
  stats: '<path d="M6 18v-5"/><path d="M12 18V8"/><path d="M18 18v-9"/>',
  ai: '<path d="M12 3.8 13.7 8.9 18.8 10.6 13.7 12.3 12 17.4 10.3 12.3 5.2 10.6 10.3 8.9 12 3.8z"/><path d="M19 15.6l.8 2.1 2.1.8-2.1.8-.8 2.1-.8-2.1-2.1-.8 2.1-.8.8-2.1z"/>',
  branding: '<path d="M12 3a9 9 0 0 0 0 18c1.6 0 2.3-1.2 1.5-2.3-.9-1.1-.3-2.7 1.2-2.7H16a5 5 0 0 0 5-5c0-4.4-4-8-9-8z"/><circle cx="7.5" cy="10" r="1"/><circle cx="11" cy="6.8" r="1"/><circle cx="15.3" cy="7.6" r="1"/>',
  'user-list': '<path d="M8 18v-1.5A3.5 3.5 0 0 1 11.5 13h1A3.5 3.5 0 0 1 16 16.5V18"/><circle cx="12" cy="8" r="3"/><path d="M4 18v-1.2a2.5 2.5 0 0 1 1.6-2.3"/><path d="M20 18v-1.2a2.5 2.5 0 0 0-1.6-2.3"/>',
  suspended: '<circle cx="12" cy="8.5" r="3"/><path d="M7 18v-1.5a5 5 0 0 1 10 0V18"/><path d="M5 5l14 14"/>',
  chart: '<path d="M4 19h16"/><path d="M7 16V10"/><path d="M12 16V6"/><path d="M17 16v-4"/>',
  usage: '<rect x="4" y="4" width="16" height="16" rx="3"/><path d="M8 15v-4"/><path d="M12 15V8"/><path d="M16 15v-2"/>',
  robot: '<rect x="5" y="8" width="14" height="10" rx="3"/><path d="M9 8V6a3 3 0 0 1 6 0v2"/><circle cx="9.5" cy="13" r="1"/><circle cx="14.5" cy="13" r="1"/><path d="M12 3v2"/>',
  settings: '<circle cx="12" cy="12" r="3"/><path d="M12 2v2M12 20v2M4.2 4.2l1.4 1.4M18.4 18.4l1.4 1.4M2 12h2M20 12h2M4.2 19.8l1.4-1.4M18.4 5.6l1.4-1.4"/>',
  queue: '<path d="M6 7h12"/><path d="M6 12h8"/><path d="M6 17h10"/><circle cx="18" cy="17" r="2"/>',
  tag: '<path d="M4 10.5V6a2 2 0 0 1 2-2h4.5L20 13.5 13.5 20 4 10.5z"/><circle cx="9" cy="9" r="1.2"/>',
  image: '<rect x="4" y="5" width="16" height="14" rx="2.5"/><circle cx="9" cy="10" r="1.8"/><path d="m4 16 4.5-4.5 3 3L15 11l5 5"/>',
  note: '<path d="M13.5 3H7a1.5 1.5 0 0 0-1.5 1.5v15A1.5 1.5 0 0 0 7 21h10a1.5 1.5 0 0 0 1.5-1.5V8.5L13.5 3z"/><path d="M13.5 3v5.5H19"/><path d="M9 13h6M9 16.5h6"/>',
  translate: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3a13.8 13.8 0 0 1 0 18 13.8 13.8 0 0 1 0-18z"/>',
}
</script>

<template>
  <svg
    class="admin-icon"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.5"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    v-html="paths"
  />
</template>

<style scoped>
.admin-icon { display: block; flex: 0 0 auto; }
</style>
