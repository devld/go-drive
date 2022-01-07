import { onBeforeUnmount, onMounted } from 'vue'

/**
 * @param {KeyboardEvent} e
 */
function hotKeyHandler(e) {
  for (const ev of this._hotkey.events) {
    try {
      if (ev.auxMatched(e) && ev.keyMatched(e)) {
        ev.handler(e)
      }
    } catch {
      // ignore
    }
  }
}

const genKeyMatched = (key) => {
  let keyMatched
  if (typeof key === 'string') keyMatched = (e) => e.key === key
  else if (typeof key === 'function') keyMatched = (e) => key(e)
  else if (Array.isArray(key)) keyMatched = (e) => key.includes(e.key)
  else throw new Error('Invalid key')
  return keyMatched
}

/**
 * @param {Function} cb
 * @param {string|Array<string>|Function} key
 * @param {*} param2
 */
export const useHotKey = (cb, key, { ctrl, alt, shift, el } = {}) => {
  ctrl = !!ctrl
  alt = !!alt
  shift = !!shift
  /**
   * @type {HTMLElement}
   */
  let el_

  onMounted(() => {
    if (typeof el === 'function') el_ = el()
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
        !(ctrl ^ e.ctrlKey) && !(alt ^ e.altKey) && !(shift ^ e.shiftKey),
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
