import Hammer from 'hammerjs'

const emitEvent = (vnode, el) => {
  if (vnode.componentInstance) {
    vnode.context.$emit('long-press')
  } else {
    el.dispatchEvent(new CustomEvent('long-press'))
  }
}

export default {
  bind(el, binding, vnode) {
    const timeout = +binding.value || 300
    const h = new Hammer(el)
    h.get('press').set({ time: timeout })
    h.on('press', () => {
      emitEvent(vnode, el)
    })
    el._hammer = h
  },
  unbind(el) {
    if (el._hammer) {
      el._hammer.destroy()
    }
  },
}
