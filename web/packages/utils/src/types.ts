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

export interface Stringifiable {
  toString(): string
}

export type TextLike = string | Stringifiable

export type MaybePromise<T> = T | Promise<T>

export type AnyRecord = Record<string, any>
