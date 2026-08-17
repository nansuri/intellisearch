<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AskBox from '../components/AskBox.vue'
import AppHeader from '../components/AppHeader.vue'
import { useSiteStore } from '../stores/site'
import { createAskGhost } from '../services/motion'
import { resolveLocationForQuery } from '../composables/useDeviceLocation'
import { needsLocationContext } from '../utils/locationIntent'

const router = useRouter()
const site = useSiteStore(); onMounted(() => site.load())
const askBox = ref<InstanceType<typeof AskBox> | null>(null)
const locating = ref(false)

async function onAsk(question: string) {
  const el = askBox.value?.$el
  if (el instanceof HTMLElement) {
    createAskGhost(el.getBoundingClientRect(), question)
  }
  if (needsLocationContext(question)) locating.value = true
  try {
    await resolveLocationForQuery(question)
  } finally {
    locating.value = false
  }
  router.push({ path: '/search', query: { q: question } })
}
</script>

<template>
  <main class="page-shell main-page">
    <AppHeader />
    <section class="hero">
      <div class="hero-brand"><h1>{{ site.settings?.siteName || 'Intellisearch' }}</h1></div>
      <AskBox ref="askBox" :show-prompt="false" :helper-text="locating ? 'Getting your location for nearby results…' : ''" @submit="onAsk" />
    </section>
  </main>
</template>
