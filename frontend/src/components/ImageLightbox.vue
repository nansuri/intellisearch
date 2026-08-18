<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import type { ImageItem } from '../services/api'
import { useToastStore } from '../stores/toast'

const props = defineProps<{ images: ImageItem[]; index: number }>()
const emit = defineEmits<{ close: [] }>()

const toast = useToastStore()
const current = ref(props.index)
const fullError = ref(false)
const downloading = ref(false)

const active = computed(() => props.images[current.value] ?? props.images[0])
// The full-size URL when the source provided one, falling back to the thumbnail.
const displayUrl = computed(() => active.value?.fullUrl || active.value?.thumbnailUrl || '')

function clamp(index: number): number {
  if (!props.images.length) return 0
  return (index + props.images.length) % props.images.length
}
function next() { current.value = clamp(current.value + 1); fullError.value = false }
function prev() { current.value = clamp(current.value - 1); fullError.value = false }

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
  else if (e.key === 'ArrowRight') next()
  else if (e.key === 'ArrowLeft') prev()
}

// Lock body scroll while the viewer is open.
watch(() => props.images, () => { fullError.value = false })
watch(current, () => { fullError.value = false })
onBeforeUnmount(() => { document.body.style.overflow = '' })

function open() {
  current.value = clamp(props.index)
  document.body.style.overflow = 'hidden'
  window.addEventListener('keydown', onKey)
}
watch(() => props.index, open, { immediate: true })
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))

function slugify(value: string): string {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 60) || 'image'
}

// The filename is derived from the URL's last path segment when it has an
// extension, otherwise from the image title.
function suggestedName(): string {
  try {
    const last = new URL(displayUrl.value).pathname.split('/').pop() || ''
    if (/\.(png|jpe?g|gif|webp|avif|svg|bmp|ico)(\?|$)/i.test(last)) return last
  } catch { /* relative or malformed URL — fall through */ }
  return `${slugify(active.value?.title || '')}.jpg`
}

async function download() {
  if (downloading.value || !displayUrl.value) return
  downloading.value = true
  try {
    // Cross-origin images often block blob fetches; when that happens we open
    // the image directly so the user can save it (long-press / Save image as).
    const response = await fetch(displayUrl.value, { mode: 'cors', credentials: 'omit' })
    if (!response.ok) throw new Error(`status ${response.status}`)
    const blob = await response.blob()
    const objectURL = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = objectURL
    anchor.download = suggestedName()
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    window.setTimeout(() => URL.revokeObjectURL(objectURL), 4000)
    toast.success('Image downloaded.')
  } catch {
    window.open(displayUrl.value, '_blank', 'noopener')
    toast.error('Direct download was blocked — opened the image instead.')
  } finally {
    downloading.value = false
  }
}
</script>

<template>
  <div class="lightbox" role="dialog" aria-modal="true" aria-label="Image viewer" @mousedown.self="emit('close')">
    <header class="lightbox-top">
      <div class="lightbox-caption">
        <strong>{{ active?.title || 'Image' }}</strong>
        <span v-if="active?.source">{{ active.source }}</span>
      </div>
      <div class="lightbox-actions">
        <a
          v-if="active?.url"
          class="lightbox-link"
          :href="active.url"
          target="_blank"
          rel="noopener noreferrer"
        >View source page</a>
        <button type="button" class="lightbox-btn" :disabled="downloading" @click="download">
          {{ downloading ? 'Preparing…' : 'Download' }}
        </button>
        <button type="button" class="lightbox-close" aria-label="Close viewer" @click="emit('close')">&#215;</button>
      </div>
    </header>

    <button
      v-if="images.length > 1"
      type="button"
      class="lightbox-nav lightbox-nav--prev"
      aria-label="Previous image"
      @click="prev"
    >&#8249;</button>

    <div class="lightbox-stage" @dblclick="next">
      <img
        v-if="displayUrl && !fullError"
        :src="displayUrl"
        :alt="active?.title || 'Image'"
        referrerpolicy="no-referrer"
        @error="fullError = true"
      />
      <div v-else class="lightbox-broken">
        <p>This image couldn't be loaded.</p>
        <a v-if="active?.url" :href="active.url" target="_blank" rel="noopener noreferrer">Open the source page instead</a>
        <a v-if="displayUrl" :href="displayUrl" target="_blank" rel="noopener noreferrer">Open the image directly</a>
      </div>
    </div>

    <button
      v-if="images.length > 1"
      type="button"
      class="lightbox-nav lightbox-nav--next"
      aria-label="Next image"
      @click="next"
    >&#8250;</button>

    <footer v-if="images.length > 1" class="lightbox-foot">
      <span class="lightbox-count">{{ current + 1 }} / {{ images.length }}</span>
      <button type="button" class="lightbox-thumb" v-for="(image, i) in images" :key="image.position" :class="{ 'lightbox-thumb--active': i === current }" :aria-label="`Show image ${i + 1}`" @click="current = i; fullError = false">
        <img :src="image.thumbnailUrl" :alt="image.title" loading="lazy" referrerpolicy="no-referrer" @error="($event.target as HTMLImageElement).style.visibility = 'hidden'" />
      </button>
    </footer>
  </div>
</template>

<style scoped>
.lightbox {
  position: fixed;
  inset: 0;
  z-index: 70;
  display: grid;
  grid-template-rows: auto 1fr auto;
  background: rgba(4, 7, 14, .92);
  backdrop-filter: blur(6px);
  color: #fff;
  animation: lightbox-in .16s ease;
}
@keyframes lightbox-in { from { opacity: 0; } }

.lightbox-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px clamp(16px, 3vw, 28px);
}
.lightbox-caption { display: grid; gap: 2px; min-width: 0; }
.lightbox-caption strong {
  overflow: hidden;
  font-size: .92rem;
  font-weight: 660;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.lightbox-caption span { color: rgba(255, 255, 255, .62); font-size: .74rem; }
.lightbox-actions { display: flex; align-items: center; gap: 10px; flex: 0 0 auto; }
.lightbox-link { color: rgba(255, 255, 255, .82); font-size: .78rem; font-weight: 620; text-decoration: none; }
.lightbox-link:hover { text-decoration: underline; color: #fff; }
.lightbox-btn {
  min-height: 34px;
  padding: 0 14px;
  border: 1px solid rgba(255, 255, 255, .28);
  border-radius: 9px;
  background: rgba(255, 255, 255, .1);
  color: #fff;
  font-size: .78rem;
  font-weight: 700;
  cursor: pointer;
  transition: background .15s ease, border-color .15s ease;
}
.lightbox-btn:hover:not(:disabled) { background: rgba(255, 255, 255, .18); border-color: rgba(255, 255, 255, .5); }
.lightbox-btn:disabled { opacity: .55; cursor: not-allowed; }
.lightbox-close {
  width: 34px;
  height: 34px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: rgba(255, 255, 255, .8);
  font-size: 1.3rem;
  line-height: 1;
  cursor: pointer;
}
.lightbox-close:hover { background: rgba(255, 255, 255, .12); color: #fff; }

.lightbox-stage {
  display: grid;
  place-items: center;
  min-height: 0;
  padding: 12px clamp(56px, 8vw, 90px) 8px;
}
.lightbox-stage img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 8px;
  box-shadow: 0 24px 70px rgba(0, 0, 0, .55);
}
.lightbox-broken { display: grid; gap: 8px; justify-items: center; color: rgba(255, 255, 255, .75); font-size: .9rem; text-align: center; }
.lightbox-broken p { margin: 0 0 6px; }
.lightbox-broken a { color: #fff; font-weight: 650; }
.lightbox-broken a:hover { text-decoration: underline; }

.lightbox-nav {
  position: absolute;
  top: 50%;
  z-index: 2;
  transform: translateY(-50%);
  width: 44px;
  height: 44px;
  border: 1px solid rgba(255, 255, 255, .18);
  border-radius: 50%;
  background: rgba(255, 255, 255, .08);
  color: #fff;
  font-size: 1.6rem;
  line-height: 1;
  cursor: pointer;
}
.lightbox-nav:hover { background: rgba(255, 255, 255, .18); }
.lightbox-nav--prev { left: 14px; }
.lightbox-nav--next { right: 14px; }

.lightbox-foot {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px clamp(16px, 3vw, 28px) 16px;
  overflow-x: auto;
}
.lightbox-count { margin-right: 6px; color: rgba(255, 255, 255, .6); font-size: .72rem; font-variant-numeric: tabular-nums; white-space: nowrap; }
.lightbox-thumb {
  width: 52px;
  height: 42px;
  flex: 0 0 auto;
  padding: 0;
  border: 2px solid transparent;
  border-radius: 8px;
  overflow: hidden;
  background: rgba(255, 255, 255, .08);
  cursor: pointer;
  transition: border-color .14s ease;
}
.lightbox-thumb img { width: 100%; height: 100%; object-fit: cover; display: block; }
.lightbox-thumb--active { border-color: #fff; }

@media (max-width: 520px) {
  .lightbox-stage { padding-inline: 52px; }
  .lightbox-link { display: none; }
}
</style>
