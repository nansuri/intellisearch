<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ImageItem } from '../services/api'
import ImageLightbox from './ImageLightbox.vue'

const props = defineProps<{ images: ImageItem[] }>()

// Thumbnails that failed to load are dropped from the grid (and its count).
const broken = ref<Set<number>>(new Set())
const visible = computed(() => props.images.filter((image) => !broken.value.has(image.position)))
const lightboxIndex = ref<number | null>(null)

function onThumbError(image: ImageItem) {
  broken.value = new Set(broken.value).add(image.position)
  // If the failing image is the one being viewed, close the viewer.
  if (lightboxIndex.value !== null && visible.value[lightboxIndex.value]?.position === image.position) {
    lightboxIndex.value = null
  }
}

function openLightbox(index: number) {
  lightboxIndex.value = index
}
</script>

<template>
  <section v-if="visible.length" class="image-grid" aria-label="Image results">
    <div class="image-grid-head">
      <div class="section-label">Images</div>
      <span class="image-grid-count">{{ visible.length }} found</span>
    </div>
    <div class="image-grid-cards">
      <button
        v-for="(image, index) in visible"
        :key="image.position"
        type="button"
        class="image-card"
        :title="image.title"
        @click="openLightbox(index)"
      >
        <span class="image-card-frame">
          <img
            :src="image.thumbnailUrl"
            :alt="image.title"
            loading="lazy"
            referrerpolicy="no-referrer"
            @error="onThumbError(image)"
          />
        </span>
        <span class="image-card-title">{{ image.title }}</span>
      </button>
    </div>

    <ImageLightbox
      v-if="lightboxIndex !== null && visible[lightboxIndex]"
      :images="visible"
      :index="lightboxIndex"
      @close="lightboxIndex = null"
    />
  </section>
</template>

<style scoped>
.image-grid { display: grid; gap: 14px; margin-top: 30px; }
.image-grid-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.image-grid-count { color: var(--color-muted); font-size: .74rem; }
.image-grid-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
}
.image-card {
  display: grid;
  gap: 8px;
  min-width: 0;
  padding: 8px;
  border: 1px solid var(--color-border);
  border-radius: 14px;
  background: var(--color-surface);
  color: inherit;
  text-align: left;
  cursor: zoom-in;
  transition: border-color .16s ease, transform .16s ease, box-shadow .16s ease;
}
.image-card:hover {
  border-color: color-mix(in srgb, var(--color-primary) 45%, var(--color-border));
  box-shadow: 0 10px 26px var(--color-shadow);
  transform: translateY(-2px);
}
.image-card-frame {
  display: block;
  overflow: hidden;
  aspect-ratio: 4 / 3;
  border-radius: 10px;
  background: var(--color-surface-subtle);
}
.image-card-frame img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.image-card-title {
  overflow: hidden;
  color: var(--color-text);
  font-size: .78rem;
  font-weight: 620;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}
@media (max-width: 480px) {
  .image-grid-cards { grid-template-columns: repeat(auto-fill, minmax(110px, 1fr)); gap: 10px; }
}
</style>
