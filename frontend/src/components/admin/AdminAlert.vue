<script setup lang="ts">
defineProps<{
  title: string
  message: string
  tone?: 'warning' | 'danger' | 'info'
  to?: string
  actionLabel?: string
}>()
</script>

<template>
  <article class="admin-alert" :class="`admin-alert--${tone || 'warning'}`">
    <div class="admin-alert-icon" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 8v5" /><path d="M12 16.5h.01" /><path d="M10.3 4.5h3.4L20 18H4L10.3 4.5z" />
      </svg>
    </div>
    <div class="admin-alert-copy">
      <strong>{{ title }}</strong>
      <p>{{ message }}</p>
    </div>
    <router-link v-if="to" class="admin-alert-action" :to="to">{{ actionLabel || 'View details' }}</router-link>
  </article>
</template>
