<script setup lang="ts">
// Live preview of the public header brand — the configured logo (or the
// initials fallback) plus site name and tagline — shown on the branding pages
// so the owner sees exactly how the identity will look in the site header.
withDefaults(defineProps<{ siteName?: string; tagline?: string; logoUrl?: string | null }>(), { siteName: '', tagline: '', logoUrl: null })
</script>

<template>
  <div class="brand-preview">
    <div class="brand-preview-bar">
      <span class="brand-preview-brand">
        <img v-if="logoUrl" class="brand-preview-logo" :src="logoUrl" alt="" />
        <span v-else class="brand-preview-mark">{{ (siteName || 'AI').slice(0, 2).toUpperCase() }}</span>
        <span class="brand-preview-copy">
          <strong>{{ siteName || 'Your site name' }}</strong>
          <span>{{ tagline || 'Your tagline appears here.' }}</span>
        </span>
      </span>
      <span class="brand-preview-actions" aria-hidden="true">
        <span class="brand-preview-dot" />
        <span class="brand-preview-orb" />
      </span>
    </div>
  </div>
</template>

<style scoped>
.brand-preview { padding: 22px; border: 1px solid var(--color-border); border-radius: 16px; background: var(--color-bg); }
.brand-preview-bar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 13px 16px; border: 1px solid var(--color-border); border-radius: 12px; background: color-mix(in srgb, var(--color-surface) 82%, transparent); }
.brand-preview-brand { display: flex; align-items: center; gap: 11px; min-width: 0; }
.brand-preview-logo { width: 38px; height: 38px; flex: 0 0 auto; border: 1px solid var(--color-border); border-radius: 10px; object-fit: contain; background: var(--color-surface-subtle); }
.brand-preview-mark { display: inline-grid; place-items: center; width: 38px; height: 38px; flex: 0 0 auto; border-radius: 10px; background: linear-gradient(135deg, var(--color-primary), color-mix(in srgb, var(--color-primary) 55%, #8b6dff)); color: var(--color-primary-contrast); font-size: .8rem; font-weight: 760; letter-spacing: -.04em; }
.brand-preview-copy { display: grid; gap: 1px; min-width: 0; }
.brand-preview-copy strong { font-size: .95rem; letter-spacing: -.02em; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.brand-preview-copy span { color: var(--color-muted); font-size: .78rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.brand-preview-actions { display: flex; align-items: center; gap: 8px; }
.brand-preview-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--color-surface-subtle); }
.brand-preview-orb { width: 26px; height: 26px; border-radius: 50%; border: 1px solid var(--color-border); background: var(--color-surface-subtle); }
</style>
