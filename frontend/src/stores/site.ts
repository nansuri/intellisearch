import { defineStore } from 'pinia'
import { getSite, type SiteSettings } from '../services/api'

// Keeps the browser tab in sync with the configured branding: the document
// title and favicon follow site_settings instead of the static index.html.
function applyBranding(settings: SiteSettings | null) {
  if (!settings) return
  document.title = settings.siteName || 'Intellisearch'
  const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (link) link.href = settings.faviconUrl || '/favicon.svg'
}

export const useSiteStore = defineStore('site', {
  state: () => ({ settings: null as SiteSettings | null }),
  actions: {
    async load() {
      this.settings = await getSite()
      applyBranding(this.settings)
    },
  },
})

