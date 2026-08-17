const GHOST_ATTR = 'data-ask-ghost'

// createAskGhost pins a fixed copy of the ask bar at its main-page position
// before navigation, so the bar visually stays put while the page fades out.
export function createAskGhost(from: DOMRect, text: string): void {
  const ghost = document.createElement('div')
  ghost.setAttribute(GHOST_ATTR, '')
  ghost.className = 'ask-ghost'
  ghost.style.left = `${from.left}px`
  ghost.style.top = `${from.top}px`
  ghost.style.width = `${from.width}px`
  ghost.style.height = `${from.height}px`

  const icon = document.createElementNS('http://www.w3.org/2000/svg', 'svg')
  icon.setAttribute('viewBox', '0 0 24 24')
  icon.setAttribute('aria-hidden', 'true')
  icon.classList.add('ask-ghost-icon')
  const circle = document.createElementNS('http://www.w3.org/2000/svg', 'circle')
  circle.setAttribute('cx', '11'); circle.setAttribute('cy', '11'); circle.setAttribute('r', '7')
  circle.setAttribute('fill', 'none'); circle.setAttribute('stroke', 'currentColor'); circle.setAttribute('stroke-width', '2')
  const path = document.createElementNS('http://www.w3.org/2000/svg', 'path')
  path.setAttribute('d', 'M16.5 16.5L21 21')
  path.setAttribute('stroke', 'currentColor'); path.setAttribute('stroke-width', '2'); path.setAttribute('stroke-linecap', 'round')
  icon.append(circle, path)

  const label = document.createElement('span')
  label.classList.add('ask-ghost-text')
  label.textContent = text

  ghost.append(icon, label)
  document.body.appendChild(ghost)
}

// settleAskGhost finds the ghost created by createAskGhost and animates it from
// the main-page position to the results-page header bar, then removes it.
export function settleAskGhost(): void {
  const ghost = document.querySelector(`[${GHOST_ATTR}]`) as HTMLElement | null
  if (!ghost) return
  const target = document.querySelector('.app-header .ask-box') as HTMLElement | null
  const remove = () => ghost.remove()
  if (!target) { remove(); return }

  const from = ghost.getBoundingClientRect()
  const to = target.getBoundingClientRect()
  const dx = to.left - from.left
  const dy = to.top - from.top
  const scaleX = to.width / from.width
  const scaleY = to.height / from.height

  requestAnimationFrame(() => {
    ghost.style.transform = `translate(${dx}px, ${dy}px) scale(${scaleX}, ${scaleY})`
    ghost.style.borderRadius = '999px'
  })
  window.setTimeout(remove, 360)
}