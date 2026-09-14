import type { ProgressLike } from './types'

export function formatTime(value: any) {
  if (!value) return ''
  const date = value instanceof Date ? value : new Date(value)
  if (isNaN(date.getTime())) return ''
  if (date.getTime() < 0) return ''
  const pad = (part: number) => (part < 10 ? '0' + part : part)
  const month = pad(date.getMonth() + 1)
  const day = pad(date.getDate())
  const hour = pad(date.getHours())
  const minute = pad(date.getMinutes())
  const second = pad(date.getSeconds())
  return `${date.getFullYear()}-${month}-${day} ${hour}:${minute}:${second}`
}

export function formatProgressPercent(progress?: ProgressLike) {
  if (!progress || (progress.loaded === 0 && progress.total === 0)) return ''
  const value =
    progress.total > 0
      ? Math.round((progress.loaded / progress.total) * 100)
      : 0
  return `${Math.max(0, Math.min(100, value))}%`
}

export function formatBytes(bytes: number, decimals = 2) {
  if (typeof bytes !== 'number') return ''
  if (bytes < 0) return '-'
  if (bytes === 0) return '0 B'

  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'K', 'M', 'G', 'T']

  let i = Math.floor(Math.log(bytes) / Math.log(k))
  if (i >= sizes.length) i = sizes.length - 1

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
}

export function formatPercent(n: number, based?: number) {
  if (typeof n !== 'number') return ''
  if (typeof based === 'number') {
    if (based === 0) return ''
    n /= based
  }
  return (n * 100).toFixed(1) + '%'
}
