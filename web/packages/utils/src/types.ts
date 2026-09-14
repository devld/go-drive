export interface EntryMetaLike {
  ext?: string
}

export interface EntryLike {
  name: string
  meta?: EntryMetaLike
}

export type EntryMatcherInput = string | EntryLike

export interface ProgressLike {
  loaded: number
  total: number
}

export type MaybePromise<T> = T | Promise<T>

export type AnyRecord = Record<string, any>
