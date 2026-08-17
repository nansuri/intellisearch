import { defineStore } from 'pinia'
import { ApiError, getMe, login as apiLogin, logout as apiLogout, type User } from '../services/api'

const TOKEN_KEY = 'token'

export const useAuthStore = defineStore('auth', {
  state: () => ({ token: localStorage.getItem(TOKEN_KEY) || '', user: null as User | null, loading: false }),
  getters: { isAuthed: (s) => Boolean(s.token), isSuperOwner: (s) => s.user?.role === 'super_owner' },
  actions: {
    setToken(token: string) { this.token = token; localStorage.setItem(TOKEN_KEY, token) },
    async login(email: string, password: string) {
      this.loading = true
      try {
        const { token, user } = await apiLogin(email, password)
        this.setToken(token); this.user = user
        return user
      } finally { this.loading = false }
    },
    async restore() {
      if (!this.token) return
      try { this.user = await getMe() } catch (error) {
        if (error instanceof ApiError && error.code === 'AUTH01002') this.logout()
      }
    },
    async logout() { try { await apiLogout() } catch { /* ignore */ } this.clear() },
    clear() { this.token = ''; this.user = null; localStorage.removeItem(TOKEN_KEY) },
  },
})