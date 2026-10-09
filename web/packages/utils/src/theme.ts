import { arrayRemove } from './functions'

type ColorPreferenceListener = (isDark: boolean) => void
const listeners: ColorPreferenceListener[] = []
const mediaQuery =
  typeof window !== 'undefined' && typeof window.matchMedia === 'function'
    ? window.matchMedia('(prefers-color-scheme: dark)')
    : undefined

// Prefer the element's effective theme, including explicit page overrides.
export function isDarkMode(target?: Element) {
  const element =
    target ??
    (typeof document !== 'undefined' ? document.documentElement : undefined)
  if (element) {
    const schemes = getComputedStyle(element).colorScheme.split(/\s+/)
    if (schemes.includes('dark')) return true
    if (schemes.includes('light')) return false
  }
  return mediaQuery?.matches ?? false
}

mediaQuery?.addEventListener('change', () => {
  listeners.forEach((listener) => listener(mediaQuery.matches))
})

export function addPreferColorListener(listener: ColorPreferenceListener) {
  listeners.push(listener)
}

export function removePreferColorListener(listener: ColorPreferenceListener) {
  arrayRemove(listeners, (value) => value === listener)
}

// The callback receives the effective dark mode after each theme change.
export function observeThemeChanges(
  listener: ColorPreferenceListener,
  target?: Element
) {
  const notify = () => listener(isDarkMode(target))
  addPreferColorListener(notify)

  if (typeof document === 'undefined' || typeof MutationObserver === 'undefined') {
    return () => removePreferColorListener(notify)
  }

  const observer = new MutationObserver(notify)
  observer.observe(target ?? document.documentElement, {
    attributes: true,
    attributeFilter: ['class', 'data-theme', 'style'],
  })

  return () => {
    observer.disconnect()
    removePreferColorListener(notify)
  }
}
