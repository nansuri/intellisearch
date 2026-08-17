<script setup lang="ts">
import type { ImageItem } from '../services/api'

defineProps<{ images: ImageItem[] }>()
</script>

<template>
  <section v-if="images.length" class="image-grid" aria-label="Image results">
    <div class="image-grid-head">
      <div class="section-label">Images</div>
      <span class="image-grid-count">{{ images.length }} found</span>
    </div>
    <div class="image-grid-cards">
      <a
        v-for="image in images"
        :key="image.position"
        class="image-card"
        :href="image.url"
        target="_blank"
        rel="noopener noreferrer"
        :title="image.title"
      >
        <span class="image-card-frame">
          <img :src="image.thumbnailUrl" :alt="image.title" loading="lazy" referrerpolicy="no-referrer" />
        </span>
        <span class="image-card-title">{{ image.title }}</span>
      </a>
    </div>
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
  text-decoration: none;
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
</style>
