import type { Ref } from 'vue'

const ratioFromEvent = (el: HTMLElement, event: PointerEvent) => {
  const rect = el.getBoundingClientRect()
  return Math.min(1, Math.max(0, (event.clientX - rect.left) / rect.width))
}

export const createDrag = (
  elRef: Ref<HTMLElement | undefined>,
  onChange: (ratio: number) => void
) => {
  const onMove = (event: PointerEvent) => {
    if (elRef.value) onChange(ratioFromEvent(elRef.value, event))
  }
  const onUp = () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
  }
  return (event: PointerEvent) => {
    if (!elRef.value) return
    onChange(ratioFromEvent(elRef.value, event))
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }
}

export type ResizeCallback = (event: ResizeObserverEntry) => void

interface ResizeObservedHTMLElement extends HTMLElement {
  __resizeListeners__?: ResizeCallback[]
  __ro__?: ResizeObserver
}

const resizeHandler: ResizeObserverCallback = (entries) => {
  for (const entry of entries) {
    const listeners =
      (entry.target as ResizeObservedHTMLElement).__resizeListeners__ || []
    listeners.forEach((listener) => listener(entry))
  }
}

export function addResizeListener(element: HTMLElement, listener: ResizeCallback) {
  const el = element as ResizeObservedHTMLElement
  if (!el.__resizeListeners__) {
    el.__resizeListeners__ = []
    el.__ro__ = new ResizeObserver(resizeHandler)
    el.__ro__.observe(el)
  }
  el.__resizeListeners__.push(listener)
}

export function removeResizeListener(
  element: HTMLElement,
  listener: ResizeCallback
) {
  const el = element as ResizeObservedHTMLElement
  if (!el.__resizeListeners__) return
  const index = el.__resizeListeners__.indexOf(listener)
  if (index < 0) return
  el.__resizeListeners__.splice(index, 1)
  if (!el.__resizeListeners__.length) {
    el.__ro__?.disconnect()
    delete el.__resizeListeners__
    delete el.__ro__
  }
}
