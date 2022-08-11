import { onBeforeUnmount, onMounted } from 'vue'

interface HotKey_ {
  handler: (this: HTMLElement, e: KeyboardEvent) => void
  events: {
    auxMatched: Fn1<KeyboardEvent, boolean>
    keyMatched: Fn1<KeyboardEvent, boolean>
    handler: Fn1<KeyboardEvent, void>
  }[]
}

type HotKeyHTMLElement = HTMLElement & { _hotkey?: HotKey_ }

function hotKeyHandler(this: HotKeyHTMLElement, e: KeyboardEvent) {
  if (!e.key) return
  const target = e.target as HotKeyHTMLElement
  if (
    (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') &&
    !e.ctrlKey &&
    !e.altKey &&
    !e.shiftKey &&
    e.key.length === 1
  ) {
    return
  }

  for (const ev of this._hotkey!.events) {
    try {
      if (ev.auxMatched(e) && ev.keyMatched(e)) {
        ev.handler(e)
      }
    } catch {
      // ignore
    }
  }
}

const genKeyMatched = (
  key: string | string[] | Fn1<KeyboardEvent, boolean>
) => {
  let keyMatched
  if (typeof key === 'string') keyMatched = (e: KeyboardEvent) => e.key === key
  else if (typeof key === 'function') keyMatched = (e: KeyboardEvent) => key(e)
  else if (Array.isArray(key)) {
    keyMatched = (e: KeyboardEvent) => key.includes(e.key)
  } else throw new Error('Invalid key')
  return keyMatched
}

export const useHotKey = (
  cb: Fn1<KeyboardEvent, void>,
  key: string | string[] | Fn1<KeyboardEvent, boolean>,
  {
    ctrl,
    alt,
    shift,
    el,
  }: {
    ctrl?: boolean
    alt?: boolean
    shift?: boolean
    el?: HTMLElement | (() => HTMLElement)
  } = {}
) => {
  ctrl = !!ctrl
  alt = !!alt
  shift = !!shift

  let el_: HotKeyHTMLElement

  onMounted(() => {
    if (typeof el === 'function') el_ = el() as HotKeyHTMLElement
    el_ = el_ || window

    if (!el_._hotkey) {
      el_._hotkey = {
        handler: hotKeyHandler.bind(el_),
        events: [],
      }
      el_.addEventListener('keydown', el_._hotkey.handler)
    }

    el_._hotkey.events.push({
      auxMatched: (e) =>
        !(+ctrl! ^ +e.ctrlKey) &&
        !(+alt! ^ +e.altKey!) &&
        !(+shift! ^ +e.shiftKey),
      keyMatched: genKeyMatched(key),
      handler: cb,
    })
  })

  onBeforeUnmount(() => {
    if (!el_._hotkey) return

    const index = el_._hotkey.events.findIndex((ev) => ev.handler === cb)
    if (index >= 0) {
      el_._hotkey.events.splice(index, 1)
    }
    if (el_._hotkey.events.length === 0) {
      el_.removeEventListener('keydown', el_._hotkey.handler)
      delete el_._hotkey
    }
  })
}
