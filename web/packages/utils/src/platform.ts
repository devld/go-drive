export const isMacOS = () =>
  typeof navigator !== 'undefined' &&
  /Macintosh|Mac OS X/.test(navigator.userAgent)
