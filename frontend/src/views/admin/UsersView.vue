<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { createUser, deleteUser, listUsers, updateUser, type User } from '../../services/api'
import { useToastStore } from '../../stores/toast'
import PageHeader from '../../components/PageHeader.vue'
import LoadingSpinner from '../../components/LoadingSpinner.vue'
import PaginationBar from '../../components/PaginationBar.vue'
import EmptyState from '../../components/EmptyState.vue'
import ConfirmModal from '../../components/ConfirmModal.vue'
import UserFormModal from './UserFormModal.vue'
import UserTable from './UserTable.vue'

const props = defineProps<{ status?: 'suspended' }>()
const toast = useToastStore()
const users = ref<User[]>([])
const total = ref(0)
const page = ref(1)
const q = ref('')
const pageSize = 20
const loading = ref(false)
let debounce: ReturnType<typeof setTimeout> | undefined

const modalOpen = ref(false)
const editing = ref<User | null>(null)
const confirm = ref<{ action: 'suspend' | 'activate' | 'delete'; user: User } | null>(null)
const busy = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await listUsers(q.value.trim(), page.value, pageSize)
    users.value = props.status === 'suspended' ? res.users.filter((u) => u.status === 'suspended') : res.users
    total.value = res.total
  } catch (e) { toast.error((e as Error).message) } finally { loading.value = false }
}
function onSearch() { clearTimeout(debounce); debounce = setTimeout(() => { page.value = 1; load() }, 300) }
function onVisible() { if (!document.hidden) load() }
watch(page, load)
onMounted(() => { load(); document.addEventListener('visibilitychange', onVisible) })
onUnmounted(() => document.removeEventListener('visibilitychange', onVisible))

function openCreate() { editing.value = null; modalOpen.value = true }
function openEdit(user: User) { editing.value = user; modalOpen.value = true }
async function save(payload: { name: string; email: string; password?: string; role: string; aiDailyQuota: number }) {
  try {
    if (editing.value) await updateUser(editing.value.id, { ...payload, status: editing.value.status })
    else await createUser({ ...payload, password: payload.password! })
    modalOpen.value = false; toast.success(editing.value ? 'User updated.' : 'User created.')
    load()
  } catch (e) { toast.error((e as Error).message) }
}
async function runConfirm() {
  if (!confirm.value) return
  const { action, user } = confirm.value
  busy.value = true
  try {
    if (action === 'delete') { await deleteUser(user.id); toast.success('User deleted.') }
    else { await updateUser(user.id, { status: action === 'suspend' ? 'suspended' : 'active', aiDailyQuota: user.aiDailyQuota }); toast.success(action === 'suspend' ? 'User suspended.' : 'User reinstated.') }
    confirm.value = null; load()
  } catch (e) { toast.error((e as Error).message) } finally { busy.value = false }
}
</script>
<template>
  <div>
    <PageHeader :eyebrow="status === 'suspended' ? 'Users / status' : 'Users'" :title="status === 'suspended' ? 'Suspended users' : 'All users'" :description="status === 'suspended' ? 'Accounts you have suspended and can reinstate at any time.' : 'Search, create, and manage every account on the platform.'">
      <button v-if="!status" class="base-button button-primary" @click="openCreate">New user</button>
    </PageHeader>
    <section class="admin-card">
      <div class="user-toolbar">
        <input v-model="q" class="text-input user-search" type="search" placeholder="Search by name or email…" @input="onSearch" />
        <span class="user-total">{{ total }} account{{ total === 1 ? '' : 's' }}</span>
      </div>
      <div v-if="loading" class="admin-loading"><LoadingSpinner /></div>
      <div v-else-if="users.length">
        <UserTable :users="users" @edit="openEdit" @suspend="(u) => confirm = { action: 'suspend', user: u }" @reinstate="(u) => confirm = { action: 'activate', user: u }" @delete="(u) => confirm = { action: 'delete', user: u }" />
        <PaginationBar v-if="total > pageSize" :page="page" :total-pages="Math.ceil(total / pageSize)" :total="total" @change="page = $event" />
      </div>
      <EmptyState v-else :title="status === 'suspended' ? 'No suspended accounts' : (q ? 'No accounts found' : 'No users yet')" :message="status === 'suspended' ? 'Suspended accounts will appear here.' : q ? 'Try a different search.' : 'Create the first account to get started.'">
        <button v-if="!status" class="base-button button-secondary" @click="openCreate">New user</button>
        <button v-else class="base-button button-secondary" @click="q = ''; load()">Clear search</button>
      </EmptyState>
    </section>
    <UserFormModal :open="modalOpen" :user="editing" @close="modalOpen = false" @save="save" />
    <ConfirmModal v-if="confirm" :open="true" :title="confirm.action === 'delete' ? 'Delete user' : confirm.action === 'suspend' ? 'Suspend user' : 'Reinstate user'" :message="confirm.action === 'delete' ? `Delete ${confirm.user.email} permanently? Their history and sessions are removed.` : confirm.action === 'suspend' ? `Suspend ${confirm.user.email}? They can no longer sign in until reinstated.` : `Reinstate ${confirm.user.email}?`" confirm-label="Confirm" :busy="busy" @close="confirm = null" @confirm="runConfirm" />
  </div>
</template>