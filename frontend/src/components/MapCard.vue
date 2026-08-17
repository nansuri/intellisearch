<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { MapPoint } from '../services/api'

const props = withDefaults(defineProps<{ center?: MapPoint | null; markers?: MapPoint[] }>(), {
  center: null,
  markers: () => [],
})

const container = ref<HTMLDivElement | null>(null)
let mapInstance: any = null
let layerGroup: any = null
let disposed = false

// Default tile layer; override via VITE_MAP_TILE_URL for self-hosted tiles.
const TILE_URL = import.meta.env.VITE_MAP_TILE_URL || 'https://tile.openstreetmap.org/{z}/{x}/{y}.png'

const hasData = computed(() => Boolean(props.center) || props.markers.length > 0)
const externalUrl = computed(() => {
  const focus = props.center || props.markers[0]
  if (!focus) return 'https://www.openstreetmap.org/'
  return `https://www.openstreetmap.org/?mlat=${focus.latitude}&mlon=${focus.longitude}#map=14/${focus.latitude}/${focus.longitude}`
})

async function renderMap() {
  if (!container.value || disposed || !hasData.value) return
  if (!mapInstance) {
    const L = (await import('leaflet')).default
    await import('leaflet/dist/leaflet.css')
    if (!container.value || disposed) return
    mapInstance = L.map(container.value, { scrollWheelZoom: false })
    L.tileLayer(TILE_URL, { maxZoom: 19, attribution: '&copy; OpenStreetMap contributors' }).addTo(mapInstance)
    layerGroup = L.layerGroup().addTo(mapInstance)
  }
  layerGroup.clearLayers()
  const L = mapInstance.constructor

  const points: Array<[number, number]> = []
  const userIcon = L.divIcon({ className: 'map-pin map-pin--you', html: '<span></span>', iconSize: [16, 16], iconAnchor: [8, 8] })
  if (props.center) {
    L.marker([props.center.latitude, props.center.longitude], { icon: userIcon, zIndexOffset: 1000 })
      .addTo(layerGroup)
      .bindPopup(props.center.label || 'You are here')
    points.push([props.center.latitude, props.center.longitude])
  }
  props.markers.forEach((marker, index) => {
    const icon = L.divIcon({ className: 'map-pin', html: `<span>${index + 1}</span>`, iconSize: [24, 24], iconAnchor: [12, 12] })
    L.marker([marker.latitude, marker.longitude], { icon })
      .addTo(layerGroup)
      .bindPopup(marker.label || `Result ${index + 1}`)
    points.push([marker.latitude, marker.longitude])
  })

  if (points.length > 1) {
    mapInstance.fitBounds(points, { padding: [36, 36], maxZoom: 15 })
  } else if (points.length === 1) {
    mapInstance.setView(points[0], 14)
  }
}

function invalidate() {
  if (mapInstance) window.setTimeout(() => mapInstance.invalidateSize(), 0)
}

onMounted(() => {
  void renderMap()
  invalidate()
})
onBeforeUnmount(() => {
  disposed = true
  if (mapInstance) {
    mapInstance.remove()
    mapInstance = null
    layerGroup = null
  }
})
watch(() => [props.center?.latitude, props.center?.longitude, props.markers.length], () => {
  if (hasData.value) void renderMap()
})
</script>

<template>
  <section v-if="hasData" class="map-card" aria-label="Map of nearby results">
    <div class="map-card-head">
      <div>
        <div class="section-label">Map</div>
        <p v-if="center" class="map-card-center">{{ center.label }}</p>
        <p v-else class="map-card-center">{{ markers.length }} nearby {{ markers.length === 1 ? 'place' : 'places' }}</p>
      </div>
      <a class="map-card-open" :href="externalUrl" target="_blank" rel="noopener noreferrer">Open in OpenStreetMap</a>
    </div>
    <div ref="container" class="map-card-canvas" />
    <p v-if="markers.length" class="map-card-note">{{ markers.length }} nearby {{ markers.length === 1 ? 'result' : 'results' }} shown on the map.</p>
  </section>
</template>

<style scoped>
.map-card { display: grid; gap: 12px; margin-top: 26px; padding: 16px; border: 1px solid var(--color-border); border-radius: 16px; background: var(--color-surface); }
.map-card-head { display: flex; align-items: start; justify-content: space-between; gap: 14px; }
.map-card-center { margin: 4px 0 0; color: var(--color-muted); font-size: .82rem; line-height: 1.45; }
.map-card-open { flex: 0 0 auto; color: var(--color-primary); font-size: .76rem; font-weight: 680; text-decoration: none; }
.map-card-open:hover { text-decoration: underline; }
.map-card-canvas { height: 280px; overflow: hidden; border-radius: 12px; border: 1px solid var(--color-border); background: #dfe7ee; z-index: 0; }
.map-card-note { margin: 0; color: var(--color-muted); font-size: .74rem; }
@media (max-width: 520px) { .map-card-canvas { height: 220px; } .map-card-head { flex-direction: column; } }
</style>

<style>
/* Leaflet-injected DOM is outside Vue's scoped styles, so pins are global. */
.map-pin {
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: var(--color-primary);
  color: #fff;
  font-size: .66rem;
  font-weight: 800;
  box-shadow: 0 3px 8px rgba(0, 0, 0, .3);
}
.map-pin--you {
  background: var(--color-text);
  border: 2px solid #fff;
}
.map-pin--you span {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #fff;
}
</style>
