<script setup lang="ts">
import { computed } from 'vue'
import { useSiteStore } from '../stores/site'

const site = useSiteStore()

// Copyright text comes from site_settings (Owner Control Panel → Branding →
// Identity). Falls back to the site name, then the generic default.
const label = computed(() => site.settings?.copyright?.trim() || site.settings?.siteName?.trim() || 'Intellisearch')
const year = new Date().getFullYear()
</script>

<template>
  <footer class="site-footer">
    <span>© {{ year }} {{ label }}</span>
  </footer>
</template>

<style scoped>
.site-footer {
  display: flex;
  justify-content: center;
  padding: 26px 20px 34px;
  color: var(--color-muted);
  font-size: .74rem;
  text-align: center;
}
</style>
