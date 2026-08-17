<script setup lang="ts">
import type { User } from '../../services/api'
import Avatar from '../../components/Avatar.vue'
import StatusBadge from '../../components/StatusBadge.vue'

defineProps<{ users: User[] }>()
const emit = defineEmits<{ (e: 'edit', user: User): void; (e: 'suspend', user: User): void; (e: 'reinstate', user: User): void; (e: 'delete', user: User): void }>()

const roleOptions: Record<string, { label: string; tone?: string }> = { general_user: { label: 'User' }, super_owner: { label: 'Owner', tone: 'accent' } }
const statusOptions: Record<string, { label: string; tone: string }> = { active: { label: 'Active', tone: 'success' }, suspended: { label: 'Suspended', tone: 'danger' } }
const fmt = (iso: string | null) => (iso ? new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' }) : '—')
</script>
<template>
  <div class="user-table-wrap">
    <table class="user-table">
      <thead><tr><th>User</th><th>Role</th><th>Status</th><th>Quota</th><th>Last login</th><th></th></tr></thead>
      <tbody>
        <tr v-for="u in users" :key="u.id">
          <td><span class="cell-user"><Avatar :name="u.name" :avatar-url="u.avatarUrl" /><span><strong>{{ u.name }}</strong><small>{{ u.email }}</small></span></span></td>
          <td><StatusBadge :value="u.role" :options="roleOptions" /></td>
          <td><StatusBadge :value="u.status" :options="statusOptions" /></td>
          <td class="cell-num">{{ u.aiDailyQuota }}/day</td>
          <td class="cell-muted">{{ fmt(u.lastLoginAt) }}</td>
          <td class="cell-actions">
            <button class="table-action" @click="emit('edit', u)">Edit</button>
            <button v-if="u.status === 'active'" class="table-action" @click="emit('suspend', u)">Suspend</button>
            <button v-else class="table-action" @click="emit('reinstate', u)">Reinstate</button>
            <button class="table-action table-action--danger" @click="emit('delete', u)">Delete</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>