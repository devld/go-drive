import type { Directive } from 'vue'

const emitEvent = (el: HTMLElement) => {
  el.dispatchEvent(new CustomEvent('long-press'))
}

interface LongPressTarget extends HTMLElement {
  __longPress: {
    start: (e: TouchEvent) => void
    move: (e: TouchEvent) => void
    end: (e: TouchEvent) => void
  }
}

const TOUCH_THRESHOLD = 10
const TOUCH_TIME = 300

export default {
  mounted(el: LongPressTarget) {
    let x: number, y: number, timer: number
    let triggered = false
    const onStart = (e: TouchEvent) => {
      x = e.touches[0].clientX
      y = e.touches[0].clientY
      timer = setTimeout(() => {
        emitEvent(el)
        triggered = true
      }, TOUCH_TIME) as unknown as number
    }
    const onMove = (e: TouchEvent) => {
      const t = e.changedTouches[0]
      const deltaX = Math.abs(x - t.clientX)
      const deltaY = Math.abs(y - t.clientY)
      if (deltaX > TOUCH_THRESHOLD || deltaY > TOUCH_THRESHOLD) {
        clearTimeout(timer)
        return
      }
    }
    const onEnd = (e: TouchEvent) => {
      clearTimeout(timer)
      if (triggered) {
        e.preventDefault()
        triggered = false
      }
    }
    el.addEventListener('touchstart', onStart)
    el.addEventListener('touchmove', onMove)
    el.addEventListener('touchend', onEnd)
    el.__longPress = {
      start: onStart,
      move: onMove,
      end: onEnd,
    }
  },
  beforeUnmount(el: LongPressTarget) {
    el.removeEventListener('touchstart', el.__longPress.start)
    el.removeEventListener('touchmove', el.__longPress.move)
    el.removeEventListener('touchend', el.__longPress.end)
  },
} as Directive
