<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { User } from '../../services/api'
import BaseModal from '../../components/BaseModal.vue'
import FormField from '../../components/FormField.vue'

const props = defineProps<{ open: boolean; user: User | null }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'save', payload: { name: string; email: string; password?: string; role: string; aiDailyQuota: number }): void }>()
const name = ref(''); const email = ref(''); const password = ref(''); const role = ref('general_user'); const quota = ref(100)
const saving = ref(false)
const isNew = computed(() => !props.user)

watch(() => props.open, (v) => {
  if (!v) return
  name.value = props.user?.name || ''
  email.value = props.user?.email || ''
  password.value = ''
  role.value = props.user?.role || 'general_user'
  quota.value = props.user?.aiDailyQuota ?? 100
})
function submit() {
  saving.value = true
  emit('save', { name: name.value.trim(), email: email.value.trim(), ...(!isNew.value ? {} : { password: password.value }), role: role.value, aiDailyQuota: Number(quota.value) })
  setTimeout(() => (saving.value = false), 400)
}
</script>
<template>
  <BaseModal :open="open" :title="isNew ? 'New user' : 'Edit user'" @close="emit('close')">
    <form class="admin-form" @submit.prevent="submit">
      <FormField label="Full name" :error="name ? '' : undefined">
        <input v-model="name" class="text-input" required placeholder="Jane Doe" />
      </FormField>
      <FormField label="Email" :error="email ? '' : undefined">
        <input v-model="email" type="email" class="text-input" required placeholder="jane@example.com" />
      </FormField>
      <FormField v-if="isNew" label="Password" :error="password ? '' : undefined">
        <input v-model="password" type="password" class="text-input" required minlength="8" placeholder="Minimum 8 characters" />
      </FormField>
      <div class="form-grid-2">
        <FormField label="Role">
          <select v-model="role" class="text-input">
            <option value="general_user">General user</option>
            <option value="super_owner">Super owner</option>
          </select>
        </FormField>
        <FormField label="Daily AI quota" hint="Questions per day">
          <input v-model.number="quota" type="number" class="text-input" min="0" required />
        </FormField>
      </div>
      <footer v-if="!isNew" class="modal-or">Changing a role here takes effect immediately on the next request.</footer>
      <div class="modal-submit-row">
        <button type="button" class="base-button button-secondary" @click="emit('close')">Cancel</button>
        <button type="submit" class="base-button button-primary" :disabled="saving || !name || !email || (isNew && !password)">{{ saving ? 'Saving…' : isNew ? 'Create user' : 'Save changes' }}</button>
      </div>
    </form>
  </BaseModal>
</template>