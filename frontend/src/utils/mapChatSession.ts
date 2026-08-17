import type { ChatSession, Source } from '../services/api'
import type { FollowUpEntry } from '../components/FollowUpBlock.vue'

/** Maps a stored chat session into result-page state. */
export function mapChatSession(session: ChatSession): {
  answer: string
  sources: Source[]
  thread: FollowUpEntry[]
  followUpSeq: number
} {
  const completed = session.messages.filter((message) => message.status === 'completed')
  let answer = ''
  let sources: Source[] = []
  const thread: FollowUpEntry[] = []
  let followUpSeq = 0

  for (let index = 0; index < completed.length - 1; index++) {
    const userMessage = completed[index]
    if (userMessage.role !== 'user') continue
    const assistant = completed[index + 1]
    if (assistant.role !== 'assistant') continue

    if (!answer) {
      answer = assistant.content
      sources = assistant.sources || []
    } else {
      followUpSeq += 1
      thread.push({
        id: followUpSeq,
        question: userMessage.content,
        answer: assistant.content,
        sources: assistant.sources || [],
        error: null,
        loading: false,
        collapsed: false,
        highlighted: false,
      })
    }
    index += 1
  }

  return { answer, sources, thread, followUpSeq }
}
