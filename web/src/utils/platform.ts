type ModifierEvent = Pick<KeyboardEvent, 'ctrlKey' | 'metaKey'>

export const isMacOS = () =>
  typeof navigator !== 'undefined' &&
  /Macintosh|Mac OS X/.test(navigator.userAgent)

export const isPrimaryModifierPressed = (event: ModifierEvent) =>
  isMacOS()
    ? event.metaKey && !event.ctrlKey
    : event.ctrlKey && !event.metaKey
