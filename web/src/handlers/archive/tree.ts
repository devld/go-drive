import { ArchiveEntry } from '@/api/artifact'

function parentPath(path: string) {
  const index = path.lastIndexOf('/')
  return index < 0 ? '' : path.slice(0, index)
}

function virtualDir(path: string): ArchiveEntry {
  const index = path.lastIndexOf('/')
  return {
    path,
    name: index < 0 ? path : path.slice(index + 1),
    type: 'dir',
    size: 0,
    modTime: 0,
  }
}

function compareEntries(a: ArchiveEntry, b: ArchiveEntry) {
  if (a.type !== b.type) return a.type === 'dir' ? -1 : 1
  return a.name.localeCompare(b.name)
}

function addChild(
  children: Map<string, ArchiveEntry[]>,
  parent: string,
  entry: ArchiveEntry
) {
  const list = children.get(parent)
  if (list) list.push(entry)
  else children.set(parent, [entry])
}

export function buildArchiveTree(entries: ArchiveEntry[]): Map<string, ArchiveEntry[]> {
  const nodes = new Map<string, ArchiveEntry>()
  const children = new Map<string, ArchiveEntry[]>()

  const ensureDir = (path: string) => {
    if (!path || nodes.has(path)) return
    ensureDir(parentPath(path))
    const dir = virtualDir(path)
    nodes.set(path, dir)
    addChild(children, parentPath(path), dir)
  }

  for (const item of entries) {
    if (!item.path) continue
    const parent = parentPath(item.path)
    ensureDir(parent)
    const existing = nodes.get(item.path)
    if (existing) {
      existing.name = item.name
      existing.type = item.type
      existing.size = item.size
      existing.modTime = item.modTime
      existing.mimeType = item.mimeType
      continue
    }
    nodes.set(item.path, item)
    addChild(children, parent, item)
  }

  for (const list of children.values()) {
    list.sort(compareEntries)
  }
  return children
}
