import { defineStore } from 'pinia'
import { ApiError, getMe, login as apiLogin, logout as apiLogout, register as apiRegister, type User } from '../services/api'

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
    async register(name: string, email: string, password: string) {
      this.loading = true
      try {
        const { token, user } = await apiRegister(name, email, password)
        this.setToken(token); this.user = user
        return user
      } finally { this.loading = false }
    },
    // setSession completes an OAuth (Google) sign-in: the backend already issued
    // the JWT and we just need to persist it and load the profile.
    async setSession(token: string) {
      this.setToken(token)
      this.user = await getMe()
      return this.user
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