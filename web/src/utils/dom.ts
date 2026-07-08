import type { Ref } from 'vue'

const ratioFromEvent = (el: HTMLElement, e: PointerEvent) => {
  const rect = el.getBoundingClientRect()
  return Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width))
}

export const createDrag = (
  elRef: Ref<HTMLElement | undefined>,
  onChange: (ratio: number) => void
) => {
  const onMove = (e: PointerEvent) => {
    if (elRef.value) onChange(ratioFromEvent(elRef.value, e))
  }
  const onUp = () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
  }
  return (e: PointerEvent) => {
    if (!elRef.value) return
    onChange(ratioFromEvent(elRef.value, e))
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }
}

type ResizeCallback = (e: ResizeObserverEntry) => void

interface ResizeObservedHTMLElement extends HTMLElement {
  __resizeListeners__?: ResizeCallback[]
  __ro__?: ResizeObserver
}

const resizeHandler: ResizeObserverCallback = function (entries) {
  for (const entry of entries) {
    const listeners =
      (entry.target as ResizeObservedHTMLElement).__resizeListeners__ || []
    if (listeners.length) {
      listeners.forEach((fn) => {
        fn(entry)
      })
    }
  }
}

export const addResizeListener = function (
  element: HTMLElement,
  fn: ResizeCallback
) {
  const el = element as ResizeObservedHTMLElement
  if (!el.__resizeListeners__) {
    el.__resizeListeners__ = []
    el.__ro__ = new ResizeObserver(resizeHandler)
    el.__ro__.observe(el)
  }
  el.__resizeListeners__.push(fn)
}

export const removeResizeListener = function (
  element: HTMLElement,
  fn: ResizeCallback
) {
  const el = element as ResizeObservedHTMLElement
  if (!el || !el.__resizeListeners__) return
  el.__resizeListeners__.splice(el.__resizeListeners__.indexOf(fn), 1)
  if (!el.__resizeListeners__.length) {
    el.__ro__!.disconnect()
  }
}
