import type { ChatSession, ImageItem, MapPoint, Source } from '../services/api'
import type { FollowUpEntry } from '../components/FollowUpBlock.vue'

/** Splits persisted map points (position 0 = center) into center + markers. */
function splitMapPoints(points: MapPoint[] | undefined): { center: MapPoint | null; markers: MapPoint[] } {
  const list = points || []
  const center = list.find((point) => (point as MapPoint & { position?: number }).position === 0) ?? null
  const markers = list.filter((point) => (point as MapPoint & { position?: number }).position !== 0)
  return { center, markers }
}

/** Maps a stored chat session into result-page state. */
export function mapChatSession(session: ChatSession): {
  answer: string
  sources: Source[]
  images: ImageItem[]
  mapCenter: MapPoint | null
  mapMarkers: MapPoint[]
  thread: FollowUpEntry[]
  followUpSeq: number
} {
  const completed = session.messages.filter((message) => message.status === 'completed')
  let answer = ''
  let sources: Source[] = []
  let images: ImageItem[] = []
  let mapCenter: MapPoint | null = null
  let mapMarkers: MapPoint[] = []
  const thread: FollowUpEntry[] = []
  let followUpSeq = 0

  for (let index = 0; index < completed.length - 1; index++) {
    const userMessage = completed[index]
    if (userMessage.role !== 'user') continue
    const assistant = completed[index + 1]
    if (assistant.role !== 'assistant') continue

    const entryMap = splitMapPoints(assistant.mapPoints)
    if (!answer) {
      answer = assistant.content
      sources = assistant.sources || []
      images = assistant.images || []
      mapCenter = entryMap.center
      mapMarkers = entryMap.markers
    } else {
      followUpSeq += 1
      thread.push({
        id: followUpSeq,
        question: userMessage.content,
        answer: assistant.content,
        sources: assistant.sources || [],
        images: assistant.images || [],
        mapCenter: entryMap.center,
        mapMarkers: entryMap.markers,
        error: null,
        loading: false,
        collapsed: false,
        highlighted: false,
      })
    }
    index += 1
  }

  return { answer, sources, images, mapCenter, mapMarkers, thread, followUpSeq }
}
