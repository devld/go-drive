import { arrayRemove } from '.'

export function isDarkMode() {
  return matchMedia ? matchMedia('(prefers-color-scheme: dark)').matches : false
}

const listeners: Fn[] = []
matchMedia &&
  matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    listeners.forEach((fn) => {
      fn()
    })
  })

export function addPreferColorListener(fn: Fn) {
  listeners.push(fn)
}

export function removePreferColorListener(fn: Fn) {
  arrayRemove(listeners, (e) => e === fn)
}
