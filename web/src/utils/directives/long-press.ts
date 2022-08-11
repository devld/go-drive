import Hammer from 'hammerjs'
import type { Directive } from 'vue'

const emitEvent = (el: HTMLElement) => {
  el.dispatchEvent(new CustomEvent('long-press'))
}

export default {
  mounted(el, binding) {
    const timeout = +binding.value || 300
    const h = new Hammer(el)
    h.get('press').set({ time: timeout })
    h.on('press', () => {
      emitEvent(el)
    })
    el._hammer = h
  },
  beforeUnmount(el) {
    if (el._hammer) {
      el._hammer.destroy()
    }
  },
} as Directive