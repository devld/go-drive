import { arrayRemove } from '.'

export function isDarkMode() {
  return matchMedia ? matchMedia('(prefers-color-scheme: dark)').matches : false
}

const listeners = []
matchMedia &&
  matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    listeners.forEach((fn) => {
      fn()
    })
  })

export function addPreferColorListener(fn) {
  listeners.push(fn)
}

export function removePreferColorListener(fn) {
  arrayRemove(listeners, (e) => e === fn)
}
