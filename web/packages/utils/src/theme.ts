import { arrayRemove } from './functions'

export function isDarkMode() {
  return (
    typeof window !== 'undefined' &&
    window.matchMedia('(prefers-color-scheme: dark)').matches
  )
}

type ColorPreferenceListener = () => void
const listeners: ColorPreferenceListener[] = []
const mediaQuery =
  typeof window !== 'undefined'
    ? window.matchMedia('(prefers-color-scheme: dark)')
    : undefined

mediaQuery?.addEventListener('change', () => {
  listeners.forEach((listener) => listener())
})

export function addPreferColorListener(listener: ColorPreferenceListener) {
  listeners.push(listener)
}

export function removePreferColorListener(listener: ColorPreferenceListener) {
  arrayRemove(listeners, (value) => value === listener)
}

export function observeThemeChanges(
  listener: ColorPreferenceListener,
  target?: Element
) {
  addPreferColorListener(listener)

  if (typeof document === 'undefined' || typeof MutationObserver === 'undefined') {
    return () => removePreferColorListener(listener)
  }

  const observer = new MutationObserver(listener)
  observer.observe(target ?? document.documentElement, {
    attributes: true,
    attributeFilter: ['class', 'data-theme', 'style'],
  })

  return () => {
    observer.disconnect()
    removePreferColorListener(listener)
  }
}
