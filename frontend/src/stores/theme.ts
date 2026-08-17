import { defineStore } from 'pinia'

type Theme = 'light' | 'dark' | 'system'

const STORAGE_KEY = 'theme'

function prefersDark() {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

export const useThemeStore = defineStore('theme', {
  state: () => ({
    theme: (localStorage.getItem(STORAGE_KEY) as Theme) || 'system',
  }),
  actions: {
    setTheme(theme: Theme) {
      this.theme = theme
      localStorage.setItem(STORAGE_KEY, theme)
      this.apply()
    },
    apply() {
      const dark = this.theme === 'dark' || (this.theme === 'system' && prefersDark())
      document.documentElement.dataset.theme = dark ? 'dark' : 'light'
    },
  },
})

/** Apply saved theme on boot and react to OS changes when set to System. */
export function initTheme(pinia: ReturnType<typeof import('pinia').createPinia>) {
  const theme = useThemeStore(pinia)
  theme.apply()
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (theme.theme === 'system') theme.apply()
  })
}
