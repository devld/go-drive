export interface ArchiveEntry {
  path: string
  name: string
  type: 'dir' | 'file'
  size: number
  modTime: number
  mimeType?: string
  /** Raw symlink target from the archive index. Empty when the entry is not a link. */
  link?: string
  /** Resolved archive path of link. Set only when the target stays inside the archive. */
  follow?: string
  followType?: 'dir' | 'file'
}

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
  const aDir = a.type === 'dir' || a.followType === 'dir'
  const bDir = b.type === 'dir' || b.followType === 'dir'
  if (aDir !== bDir) return aDir ? -1 : 1
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
      existing.link = item.link
      continue
    }
    nodes.set(item.path, item)
    addChild(children, parent, item)
  }

  for (const node of nodes.values()) {
    if (!node.link) continue
    const resolved = resolveArchiveLink(node.path, nodes)
    if (!resolved) continue
    const target = nodes.get(resolved)
    if (!target) continue
    node.follow = resolved
    node.followType = target.type
  }

  for (const list of children.values()) {
    list.sort(compareEntries)
  }
  return children
}

// resolveArchiveLink walks each path component so a link to another link
// lands on the final file or directory. Absolute targets, paths that leave
// the archive, and cycles stay unresolved.
function resolveArchiveLink(
  linkPath: string,
  nodes: Map<string, ArchiveEntry>
): string | undefined {
  const seen = new Set<string>()

  const resolve = (path: string): string | undefined => {
    if (seen.has(path)) return undefined
    const node = nodes.get(path)
    if (!node) return undefined
    if (!node.link) return path
    seen.add(path)
    const raw = node.link
    if (raw.startsWith('/') || raw.includes('\\') || raw.includes('\0')) {
      return undefined
    }
    const base = parentPath(path)
    const parts = (base ? `${base}/${raw}` : raw).split('/')
    const out: string[] = []
    for (const part of parts) {
      if (part === '' || part === '.') continue
      if (part === '..') {
        if (out.length === 0) return undefined
        out.pop()
        continue
      }
      out.push(part)
      const sofar = out.join('/')
      const mid = nodes.get(sofar)
      if (mid?.link) {
        const resolvedMid = resolve(sofar)
        if (!resolvedMid) return undefined
        out.splice(0, out.length, ...resolvedMid.split('/').filter(Boolean))
      }
    }
    const finalPath = out.join('/')
    if (!nodes.has(finalPath)) return undefined
    if (nodes.get(finalPath)?.link) return resolve(finalPath)
    return finalPath
  }

  return resolve(linkPath)
}

function collectDescendants(
  dirPath: string,
  childrenByParent: Map<string, ArchiveEntry[]>
): string[] {
  const result: string[] = []
  const walk = (parent: string) => {
    for (const child of childrenByParent.get(parent) ?? []) {
      result.push(child.path)
      if (child.type === 'dir') walk(child.path)
    }
  }
  walk(dirPath)
  return result
}

export function isArchivePathCovered(path: string, selected: Set<string>) {
  if (selected.has(path)) return true
  let parent = parentPath(path)
  while (parent) {
    if (selected.has(parent)) return true
    parent = parentPath(parent)
  }
  return false
}

export function isArchiveDirFullySelected(
  dirPath: string,
  selected: Set<string>,
  childrenByParent: Map<string, ArchiveEntry[]>
): boolean {
  if (selected.has(dirPath) || isArchivePathCovered(dirPath, selected)) {
    return true
  }
  const children = childrenByParent.get(dirPath) ?? []
  if (children.length === 0) return selected.has(dirPath)
  return children.every((child) =>
    child.type === 'dir'
      ? isArchiveDirFullySelected(child.path, selected, childrenByParent)
      : selected.has(child.path) || isArchivePathCovered(child.path, selected)
  )
}

export function hasArchiveDescendantSelected(
  dirPath: string,
  selected: Set<string>,
  childrenByParent: Map<string, ArchiveEntry[]>
): boolean {
  for (const child of childrenByParent.get(dirPath) ?? []) {
    if (selected.has(child.path) || isArchivePathCovered(child.path, selected)) {
      return true
    }
    if (
      child.type === 'dir' &&
      hasArchiveDescendantSelected(child.path, selected, childrenByParent)
    ) {
      return true
    }
  }
  return false
}

export function toggledArchiveSelection(
  item: ArchiveEntry,
  selected: Set<string>,
  childrenByParent: Map<string, ArchiveEntry[]>
): Set<string> {
  const next = new Set(selected)
  const covered = isArchivePathCovered(item.path, next)
  const fullySelected =
    item.type === 'dir'
      ? isArchiveDirFullySelected(item.path, next, childrenByParent)
      : covered

  if (fullySelected) {
    const ancestors: string[] = []
    let parent = parentPath(item.path)
    while (parent) {
      if (next.has(parent)) ancestors.push(parent)
      parent = parentPath(parent)
    }
    for (const ancestor of ancestors) {
      next.delete(ancestor)
      for (const path of collectDescendants(ancestor, childrenByParent)) {
        if (path === item.path || path.startsWith(`${item.path}/`)) continue
        next.add(path)
      }
    }
    next.delete(item.path)
    if (item.type === 'dir') {
      for (const path of collectDescendants(item.path, childrenByParent)) {
        next.delete(path)
      }
    }
    return new Set(mergeArchiveSelection(next, childrenByParent))
  }

  if (item.type === 'dir') {
    next.add(item.path)
    for (const path of collectDescendants(item.path, childrenByParent)) {
      next.delete(path)
    }
  } else {
    next.add(item.path)
  }
  return new Set(mergeArchiveSelection(next, childrenByParent))
}

export function mergeArchiveSelection(
  selected: Set<string>,
  childrenByParent: Map<string, ArchiveEntry[]>
): string[] {
  const working = new Set(selected)
  const visit = (parent: string) => {
    const children = childrenByParent.get(parent) ?? []
    for (const child of children) {
      if (child.type === 'dir') visit(child.path)
    }
    if (!parent || children.length === 0) return
    if (!children.every((child) => working.has(child.path))) return
    working.add(parent)
    for (const child of children) working.delete(child.path)
  }
  visit('')
  return [...working]
}
