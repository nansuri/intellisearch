<script setup lang="ts">
import BaseModal from './BaseModal.vue'
defineProps<{ open: boolean; title: string; message: string; confirmLabel?: string; busy?: boolean }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'confirm'): void }>()
</script>
<template>
  <BaseModal :open="open" :title="title" :busy="busy" @close="emit('close')">
    <p class="confirm-message">{{ message }}</p>
    <template #footer>
      <button class="base-button button-secondary" :disabled="busy" @click="emit('close')">Cancel</button>
      <button class="base-button button-danger" :disabled="busy" @click="emit('confirm')">{{ busy ? 'Working…' : confirmLabel || 'Confirm' }}</button>
    </template>
  </BaseModal>
</template>