import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import MainPage from '../views/MainPage.vue'
import ResultPage from '../views/ResultPage.vue'
import AccountSettings from '../views/AccountSettings.vue'
import AuthView from '../views/AuthView.vue'
import AuthCallback from '../views/AuthCallback.vue'

const ControlPanel = () => import('../views/admin/ControlPanel.vue')

const routes = [
  { path: '/', component: MainPage },
  { path: '/search', component: ResultPage },
  { path: '/account', component: AccountSettings, meta: { requiresAuth: true } },
  { path: '/login', component: AuthView, meta: { publicAuth: true } },
  { path: '/register', redirect: () => '/login?mode=register' },
  // Login is unified for every account type; keep the old admin path redirecting
  // so stale bookmarks and links still land on the sign-in page.
  { path: '/admin/login', redirect: '/login' },
  { path: '/auth/callback', component: AuthCallback, meta: { publicCallback: true } },
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
  // Restore the session on hard refresh: the token survives in localStorage but
  // the user object is only in memory, so it must be refetched. Previously this
  // only ran when the token was missing, which is why a refresh appeared to log
  // the user out.
  if (auth.token && !auth.user) await auth.restore()
  if (to.meta.publicCallback) return true
  if (to.meta.publicAuth) {
    if (auth.isAuthed && auth.user) return { path: '/' }
    return true
  }
  if (to.meta.requiresAdmin) {
    if (!auth.isAuthed) return { path: '/login' }
    if (!auth.user) await auth.restore()
    if (!auth.user || auth.user.role !== 'super_owner') return { path: '/' }
    return true
  }
  if (to.meta.requiresAuth) {
    if (!auth.isAuthed) return { path: '/login' }
    if (!auth.user) await auth.restore()
    if (!auth.isAuthed) return { path: '/login' }
  }
  return true
})

export default router
