import { onBeforeUnmount, onMounted } from 'vue'
import { isMacOS } from '../platform'

export type HotKeyKey =
  | string
  | readonly string[]
  | ((event: KeyboardEvent) => boolean)

export interface HotKeyOptions {
  ctrl?: boolean
  meta?: boolean
  primary?: boolean
  alt?: boolean
  shift?: boolean
  el?: HTMLElement | (() => HTMLElement)
}

export type ModifierEvent = Pick<KeyboardEvent, 'ctrlKey' | 'metaKey'>

export const isPrimaryModifierPressed = (event: ModifierEvent) =>
  isMacOS()
    ? event.metaKey && !event.ctrlKey
    : event.ctrlKey && !event.metaKey

interface HotKey {
  handler: (this: HTMLElement, event: KeyboardEvent) => void
  events: {
    auxMatched: (event: KeyboardEvent) => boolean
    keyMatched: (event: KeyboardEvent) => boolean
    handler: (event: KeyboardEvent) => void
  }[]
}

type HotKeyHTMLElement = HTMLElement & { _hotkey?: HotKey }

function hotKeyHandler(this: HotKeyHTMLElement, event: KeyboardEvent) {
  if (!event.key) return
  const target = event.target as HotKeyHTMLElement
  if (
    target.tagName === 'INPUT' ||
    target.tagName === 'TEXTAREA' ||
    target.contentEditable === 'true'
  ) {
    return
  }

  for (const hotKeyEvent of this._hotkey!.events) {
    try {
      if (
        hotKeyEvent.auxMatched(event) &&
        hotKeyEvent.keyMatched(event)
      ) {
        hotKeyEvent.handler(event)
      }
    } catch {
      // Ignore a handler so one shortcut cannot break the others.
    }
  }
}

function genKeyMatched(key: HotKeyKey) {
  if (typeof key === 'string') return (event: KeyboardEvent) => event.key === key
  if (typeof key === 'function') return (event: KeyboardEvent) => key(event)
  if (Array.isArray(key)) {
    return (event: KeyboardEvent) => key.includes(event.key)
  }
  throw new Error('Invalid key')
}

export function useHotKey(
  callback: (event: KeyboardEvent) => void,
  key: HotKeyKey,
  {
    ctrl,
    meta,
    primary,
    alt,
    shift,
    el,
  }: HotKeyOptions = {}
) {
  if (primary && (ctrl || meta)) {
    throw new Error('primary cannot be combined with ctrl or meta')
  }

  const macOS = isMacOS()
  if (primary) {
    ctrl = !macOS
    meta = macOS
  }
  ctrl = !!ctrl
  meta = !!meta
  alt = !!alt
  shift = !!shift

  let element: HotKeyHTMLElement

  onMounted(() => {
    if (typeof el === 'function') element = el() as HotKeyHTMLElement
    element = element || (window as unknown as HotKeyHTMLElement)

    if (!element._hotkey) {
      element._hotkey = {
        handler: hotKeyHandler.bind(element),
        events: [],
      }
      element.addEventListener('keydown', element._hotkey.handler)
    }

    element._hotkey.events.push({
      auxMatched: (event) =>
        !(+ctrl! ^ +event.ctrlKey) &&
        !(+meta! ^ +event.metaKey) &&
        !(+alt! ^ +event.altKey!) &&
        !(+shift! ^ +event.shiftKey),
      keyMatched: genKeyMatched(key),
      handler: callback,
    })
  })

  onBeforeUnmount(() => {
    if (!element?._hotkey) return

    const index = element._hotkey.events.findIndex(
      (event) => event.handler === callback
    )
    if (index >= 0) element._hotkey.events.splice(index, 1)
    if (element._hotkey.events.length === 0) {
      element.removeEventListener('keydown', element._hotkey.handler)
      delete element._hotkey
    }
  })
}
