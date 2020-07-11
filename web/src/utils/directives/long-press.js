
const emitEvent = (vnode, el) => {
  if (vnode.componentInstance) {
    vnode.context.$emit('long-press')
  } else {
    el.dispatchEvent(new CustomEvent('long-press'))
  }
}

export default {
  bind (el, binding, vnode) {
    const timeout = +binding.value || 700
    const moveThreshold = 10

    const longPress = {
      touchStart: e => {
        longPress.consumed = false
        const touch = e.touches[0]
        longPress.start = { x: touch.clientX, y: touch.clientY }
        longPress.t = setTimeout(() => {
          longPress.consumed = true
          e.preventDefault()
          emitEvent(vnode, el)
        }, timeout)
      },
      touchMove: e => {
        const touch = e.touches[0]
        const start = longPress.start
        const deltaX = touch.clientX - start.x
        const deltaY = touch.clientY - start.y
        const delta = Math.sqrt(deltaX * deltaX + deltaY * deltaY)
        if (delta > moveThreshold) {
          clearTimeout(longPress.t)
        }
        if (longPress.consumed) {
          e.preventDefault()
        }
      },
      touchEnd: e => {
        if (longPress.consumed) {
          e.preventDefault()
        }
        longPress.start = null
        clearTimeout(longPress.t)
      },
      start: null,
      t: null,
      consumed: false
    }
    el._longPress = longPress
    el.addEventListener('touchstart', longPress.touchStart)
    el.addEventListener('touchmove', longPress.touchMove)
    el.addEventListener('touchend', longPress.touchEnd)
  },
  unbind (el, binding, vnode) {
    if (el._longPress) {
      el.removeEventListener('touchstart', el._longPress.touchStart)
      el.removeEventListener('touchmove', el._longPress.touchMove)
      el.removeEventListener('touchend', el._longPress.touchEnd)
      clearTimeout(el._longPress.t)
    }
  }
}
