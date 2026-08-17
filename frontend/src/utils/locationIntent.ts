const LOCATION_INTENT = /\b(near my place|near me|around here|nearby|close to me|in my area|around my location|near my location|where i am|my location)\b/i

export function needsLocationContext(query: string): boolean {
  return LOCATION_INTENT.test(query.trim())
}
