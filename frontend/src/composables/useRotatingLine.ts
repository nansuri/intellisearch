import { computed, onBeforeUnmount, ref } from 'vue'

// Cycles through a list of lines (e.g. dad jokes) on a timer so loading UI
// never repeats itself. Pauses when the tab is hidden to avoid useless work.
export function useRotatingLine(lines: string[], intervalMs = 8000) {
  const index = ref(0)
  let timer: ReturnType<typeof setInterval> | null = null

  const onVisibility = () => {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
    if (!document.hidden) timer = setInterval(step, intervalMs)
  }

  const step = () => {
    index.value = (index.value + 1) % lines.length
  }

  onBeforeUnmount(() => {
    if (timer) clearInterval(timer)
    document.removeEventListener('visibilitychange', onVisibility)
  })

  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', onVisibility)
    onVisibility()
  }

  return {
    line: computed(() => lines[index.value]),
  }
}