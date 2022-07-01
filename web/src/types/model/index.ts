import { ApiError } from '../'

export * from './admin'
export * from './config'

export interface Group {
  name: string

  users?: User[]
}

export interface User {
  username: string
  groups: Group[]
}

export type TaskStatus = 'pending' | 'running' | 'done' | 'error' | 'canceled'

export interface TaskProgress {
  loaded: number
  total: number
}

export interface Task<R = any> {
  id: string
  status: TaskStatus
  progress?: TaskProgress
  result?: R
  error?: ApiError
  createdAt: string
  updatedAt: string

  name: string
  group: string
}

export type EntryType = 'dir' | 'file'

export type EntryMetaUseProxy = boolean | 'cors' | 'referrer'

export interface EntryMeta extends O<any> {
  accessKey?: string
  writable?: boolean

  /** real extension of this entry */
  ext?: string

  useProxy?: EntryMetaUseProxy
}

export interface Entry {
  type: EntryType
  name: string
  path: string
  size: number
  modTime: number
  meta: EntryMeta
}

export interface SearchHitEntry {
  ext: string
  modTime: string
  name: string
  path: string
  size: number
  type: EntryType
}

export interface SearchHitItem {
  entry: SearchHitEntry
  highlights: Record<keyof SearchHitEntry, string[]>
}

export interface SearchResult {
  items: SearchHitItem[]
  next: number
}
