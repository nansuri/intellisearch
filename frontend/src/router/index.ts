import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import MainPage from '../views/MainPage.vue'
import ResultPage from '../views/ResultPage.vue'
import AccountSettings from '../views/AccountSettings.vue'
import LoginView from '../views/LoginView.vue'

const ControlPanel = () => import('../views/admin/ControlPanel.vue')

const routes = [
  { path: '/', component: MainPage },
  { path: '/search', component: ResultPage },
  { path: '/account', component: AccountSettings, meta: { requiresAuth: true } },
  { path: '/login', component: LoginView, meta: { publicAuth: true } },
  { path: '/admin/login', component: () => import('../views/admin/AdminLogin.vue'), meta: { publicAdmin: true } },
  {
    path: '/admin',
    component: ControlPanel,
    meta: { requiresAdmin: true },
    children: [
      { path: '', component: () => import('../views/admin/AdminDashboardView.vue') },
      { path: 'users', component: () => import('../views/admin/UsersView.vue') },
      { path: 'users/suspended', component: () => import('../views/admin/SuspendedUsersView.vue') },
      { path: 'stats', component: () => import('../views/admin/StatsOverviewView.vue') },
      { path: 'stats/top', component: () => import('../views/admin/TopQueriesView.vue') },
      { path: 'stats/usage', component: () => import('../views/admin/PerUserUsageView.vue') },
      { path: 'stats/ai', component: () => import('../views/admin/AIServiceStatsView.vue') },
      { path: 'ai/providers', component: () => import('../views/admin/ProvidersView.vue') },
      { path: 'ai/queue', component: () => import('../views/admin/QueueConfigView.vue') },
      { path: 'branding/identity', component: () => import('../views/admin/IdentitySettingsView.vue') },
      { path: 'branding/logo', component: () => import('../views/admin/LogoSettingsView.vue') },
    ],
  },
]

export const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.token) await auth.restore()
  if (to.meta.publicAuth || to.meta.publicAdmin) {
    if (auth.isAuthed && auth.user) return { path: auth.user.role === 'super_owner' ? '/admin' : '/account' }
    return true
  }
  if (to.meta.requiresAdmin) {
    if (!auth.isAuthed) return { path: '/admin/login' }
    if (!auth.user) await auth.restore()
    if (!auth.user || auth.user.role !== 'super_owner') return { path: '/account' }
    return true
  }
  return true
})

export default router