import type { EntryLike, EntryMatcherInput } from './types'

export function isParentPath(path: string, parent: string) {
  if (isRootPath(path)) return false
  if (isRootPath(parent)) return true
  return path.startsWith(`${parent}/`)
}

export function dir(path: string) {
  if (!path) return ''
  const i = path.lastIndexOf('/')
  if (i === -1) return ''
  return path.substring(0, i)
}

export function filename(path: string) {
  if (!path) return ''
  const i = path.lastIndexOf('/')
  if (i === -1) return path
  return path.substring(i + 1)
}

export function filenameBase(filename?: string) {
  if (!filename) return ''
  const i = filename.lastIndexOf('.')
  if (i === -1) return filename
  return filename.substring(0, i)
}

export function filenameExt(filename?: string) {
  if (!filename) return ''
  const i = filename.lastIndexOf('.')
  if (i === -1) return ''
  return filename.substring(i + 1).toLowerCase()
}

export function pathJoin(...segments: string[]) {
  return segments.filter(Boolean).join('/').replace(/\/+/g, '/')
}

export function pathClean(path: string) {
  if (!path) return ''
  const segments = path.split('/').filter(Boolean)
  const paths: string[] = []
  segments.forEach((segment) => {
    if (segment === '.') return
    if (segment === '..') paths.pop()
    else paths.push(segment)
  })
  return paths.join('/')
}

const entryName = (entry: EntryMatcherInput) =>
  typeof entry === 'string' ? entry : entry.name

const entryMeta = (entry: EntryMatcherInput) =>
  typeof entry === 'string' ? undefined : entry.meta

export function entryMatches(
  entry: EntryMatcherInput,
  matches: string | readonly string[]
) {
  const name = entryName(entry)
  const meta = entryMeta(entry)
  const matches_ = (Array.isArray(matches) ? matches : [matches]) as string[]
  for (const match of matches_) {
    const normalizedMatch = match.toLowerCase()
    if (
      normalizedMatch.startsWith('/') &&
      name.toLowerCase() === normalizedMatch.substring(1)
    ) {
      return true
    }
    if (
      normalizedMatch.includes('.') &&
      name.toLowerCase().endsWith('.' + normalizedMatch)
    ) {
      return true
    }
    const ext = meta?.ext || filenameExt(name)
    if (normalizedMatch === ext) return true
  }
  return false
}

export function createEntryExtMatcher<T extends string = string>(
  extsMap: Record<T, string[]>
): (entry: EntryLike | string) => T | undefined {
  const extMapping: Record<string, T> = {}
  const fullNameMapping: Record<string, T> = {}
  Object.keys(extsMap).forEach((key) => {
    const value = key as T
    extsMap[value].forEach((ext) => {
      if (ext.startsWith('/')) {
        fullNameMapping[ext.substring(1).toLowerCase()] = value
      } else {
        extMapping[ext.toLowerCase()] = value
      }
    })
  })
  return (entry: EntryLike | string) => {
    const name = entryName(entry)
    const meta = entryMeta(entry)
    let value = fullNameMapping[name.toLowerCase()]
    if (!value) {
      const ext = (meta?.ext || filenameExt(name)).toLowerCase()
      value = extMapping[ext]
    }
    return value
  }
}

export function isRootPath(path: string) {
  return path === ''
}
