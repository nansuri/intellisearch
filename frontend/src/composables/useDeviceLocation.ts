import type { GeoLocation } from '../services/api'
import { needsLocationContext } from '../utils/locationIntent'

const CACHE_KEY = 'device_location_v1'
const CACHE_TTL_MS = 30 * 60 * 1000

type CachedLocation = GeoLocation & { cachedAt: number }

function readCache(): GeoLocation | undefined {
  try {
    const raw = sessionStorage.getItem(CACHE_KEY)
    if (!raw) return undefined
    const parsed = JSON.parse(raw) as CachedLocation
    if (Date.now() - parsed.cachedAt > CACHE_TTL_MS) {
      sessionStorage.removeItem(CACHE_KEY)
      return undefined
    }
    return { latitude: parsed.latitude, longitude: parsed.longitude, accuracy: parsed.accuracy }
  } catch {
    return undefined
  }
}

function writeCache(location: GeoLocation) {
  const payload: CachedLocation = { ...location, cachedAt: Date.now() }
  sessionStorage.setItem(CACHE_KEY, JSON.stringify(payload))
}

function requestDeviceLocation(): Promise<GeoLocation | undefined> {
  if (typeof navigator === 'undefined' || !navigator.geolocation) return Promise.resolve(undefined)

  return new Promise((resolve) => {
    navigator.geolocation.getCurrentPosition(
      (position) => {
        resolve({
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          accuracy: position.coords.accuracy,
        })
      },
      () => resolve(undefined),
      { enableHighAccuracy: false, timeout: 12_000, maximumAge: 300_000 },
    )
  })
}

/** Returns device location when the query needs local context. */
export async function resolveLocationForQuery(query: string): Promise<GeoLocation | undefined> {
  if (!needsLocationContext(query)) return undefined

  const cached = readCache()
  if (cached) return cached

  const location = await requestDeviceLocation()
  if (location) writeCache(location)
  return location
}

export function clearCachedLocation() {
  sessionStorage.removeItem(CACHE_KEY)
}
