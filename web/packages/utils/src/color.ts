/** Resolve a CSS color expression, including custom properties, to hex. */
export function resolveCssColor(
  color: string,
  scope?: HTMLElement
): string | undefined {
  if (typeof document === 'undefined') return undefined

  const probe = document.createElement('span')
  probe.style.color = color.trim()
  if (!probe.style.color) return undefined

  const parent = scope ?? document.documentElement
  parent.append(probe)

  let computedColor = ''
  try {
    computedColor = getComputedStyle(probe).color
  } finally {
    probe.remove()
  }

  const canvas = document.createElement('canvas')
  canvas.width = 1
  canvas.height = 1
  const context = canvas.getContext('2d', { willReadFrequently: true })
  if (!context) return undefined

  context.fillStyle = computedColor
  context.fillRect(0, 0, 1, 1)
  const [red, green, blue, alpha] = context.getImageData(0, 0, 1, 1).data
  const hex = (value: number) => value.toString(16).padStart(2, '0')

  return `#${hex(red)}${hex(green)}${hex(blue)}${alpha < 255 ? hex(alpha) : ''}`
}
