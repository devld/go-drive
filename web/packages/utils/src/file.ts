import type { MaybePromise } from './types'

export interface ResolvedEntry {
  type: 'file' | 'dir'
  path: string
  file?: File
  children?: ResolvedEntry[]
}

export interface ResolvedFileEntry {
  type: 'file'
  path: string
  file: File
}

export function wrapFile(file: File): ResolvedFileEntry {
  return {
    type: 'file',
    path: file.name,
    file,
  }
}

async function resolveEntry(
  entry: FileSystemEntry,
  breakpoint: () => MaybePromise<void>
): Promise<ResolvedEntry> {
  try {
    await breakpoint()
  } catch {
    throw Error('aborted')
  }
  if (entry.isFile) {
    const file = entry as FileSystemFileEntry
    return new Promise<ResolvedFileEntry>((resolve, reject) => {
      file.file(
        (value) => {
          resolve({
            type: 'file',
            file: value,
            path: entry.fullPath,
          })
        },
        reject
      )
    })
  }
  if (entry.isDirectory) {
    const directory = entry as FileSystemDirectoryEntry
    return {
      type: 'dir',
      path: entry.fullPath,
      children: await new Promise<ResolvedEntry[]>((resolve, reject) => {
        directory.createReader().readEntries(async (entries) => {
          const children: ResolvedEntry[] = []
          for (const child of entries) {
            try {
              children.push(await resolveEntry(child, breakpoint))
            } catch (error) {
              reject(error)
              return
            }
          }
          resolve(children)
        }, reject)
      }),
    }
  }
  throw new Error('unreachable')
}

export async function resolveEntries(
  entries: FileSystemEntry[],
  onProgress?: (total: number) => MaybePromise<void>
) {
  let total = 0
  const result: ResolvedEntry[] = []
  const onFound = () => {
    total += 1
    return onProgress?.(total)
  }
  for (const entry of entries) {
    result.push(await resolveEntry(entry, onFound))
  }
  return result
}

export function getFileEntries(
  entries: ResolvedEntry[],
  result: ResolvedFileEntry[] = []
): ResolvedFileEntry[] {
  for (const entry of entries) {
    if (entry.type === 'file') result.push(entry as ResolvedFileEntry)
    else if (entry.children) getFileEntries(entry.children, result)
  }
  return result
}

export function isDataTransferHasFiles(dataTransfer: DataTransfer) {
  if (Array.from(dataTransfer.types).includes('Files')) return true
  for (let i = 0; i < dataTransfer.items.length; i++) {
    if (dataTransfer.items[i].kind === 'file') return true
  }
  return false
}

export function getDataTransferFiles(dataTransfer: DataTransfer) {
  const entries: FileSystemEntry[] = []
  const files: File[] = []
  for (let i = 0; i < dataTransfer.items.length; i++) {
    const item = dataTransfer.items[i]
    if (item.kind !== 'file') continue
    const entry = item.webkitGetAsEntry()
    if (entry) entries.push(entry)
    else {
      const file = item.getAsFile()
      if (file) files.push(file)
    }
  }
  return { entries, files }
}

export function triggerDownloadFile(url: string, filename: string) {
  const link = document.createElement('a')
  link.rel = 'noreferrer noopener nofollow'
  link.target = '_blank'
  link.href = url
  link.download = filename
  link.click()
}
