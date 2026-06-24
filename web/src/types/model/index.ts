import { ApiError } from '../'

export * from './admin'
export * from './config'

/** The built-in admin group whose members are unrestricted. */
export const ADMIN_GROUP = 'admin'

export interface Group {
  name: string

  rootPath?: string

  users?: User[]
}

export interface User {
  username: string
  groups: Group[]
  /**
   * The auth provider that owns this user. Empty for local users; external
   * providers (e.g. "ldap") set their provider name. Group membership of
   * external users is managed by the provider, so it cannot be edited locally.
   */
  source?: string
}

/**
 * Auth provider sources that sync a user's group membership from the provider.
 * For these users group membership must not be edited locally. Add future
 * group-syncing providers here.
 */
export const GROUP_SYNCED_SOURCES: readonly string[] = ['ldap']

/** Whether the user's group membership is managed by an external provider. */
export const isGroupSyncedUser = (user: Pick<User, 'source'>): boolean =>
  !!user.source && GROUP_SYNCED_SOURCES.includes(user.source)

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

export interface EntryPathMeta {
  defaultSort?: string
  defaultMode?: string
  hiddenPattern?: string
}

export interface EntryMeta extends O<any> {
  accessKey?: string
  writable?: boolean
  thumbnail?: string
  mountAt?: string
  /** real extension of this entry */
  ext?: string
  pathMeta?: EntryPathMeta
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
