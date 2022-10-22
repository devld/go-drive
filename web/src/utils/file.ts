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

export function wrapFile(file: File): ResolvedEntry {
  return {
    type: 'file',
    path: file.name,
    file,
  }
}

async function resolveEntry(
  entry: FileSystemEntry,
  breakpoint: () => PromiseValue<void>
): Promise<ResolvedEntry> {
  try {
    await breakpoint()
  } catch {
    throw Error('aborted')
  }
  if (entry.isFile) {
    const file = entry as FileSystemFileEntry
    return new Promise<ResolvedEntry>((resolve, reject) => {
      file.file(async (file) => {
        resolve({
          type: 'file',
          file,
          path: entry.fullPath,
        })
      }, reject)
    })
  } else if (entry.isDirectory) {
    const dir = entry as FileSystemDirectoryEntry
    return {
      type: 'dir',
      path: entry.fullPath,
      children: await new Promise((resolve, reject) => {
        dir.createReader().readEntries(async (entries) => {
          const children = []
          for (const entry of entries) {
            try {
              children.push(await resolveEntry(entry, breakpoint))
            } catch (e) {
              return reject(e)
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
  onProgress?: (total: number) => PromiseValue<void>
) {
  let total = 0
  const result = []
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
  result?: ResolvedFileEntry[]
): ResolvedFileEntry[] {
  if (!result) result = []
  for (const e of entries) {
    if (e.type === 'file') result.push(e as ResolvedFileEntry)
    else if (e.children) {
      getFileEntries(e.children, result)
    }
  }
  return result
}

export function isDataTransferHasFiles(dt: DataTransfer) {
  for (let i = 0; i < dt.items.length; i++) {
    const item = dt.items[i]
    if (item.kind === 'file') return true
  }
  return false
}

export function getDataTransferFiles(dt: DataTransfer) {
  const items: FileSystemEntry[] = []
  for (let i = 0; i < dt.items.length; i++) {
    const item = dt.items[i]
    if (item.kind === 'file') {
      const entry = item.webkitGetAsEntry()
      if (entry) items.push(entry)
    }
  }
  return items
}

export function triggerDownloadFile(url: string, filename: string) {
  const a = document.createElement('a')
  a.rel = 'noreferrer noopener nofollow'
  a.target = '_blank'
  a.href = url
  a.download = filename
  a.click()
}
