import EntryIcon_ from './EntryIcon.vue'
import EntryLink_ from './EntryLink.vue'
import EntryItem_ from './EntryItem.vue'
import EntryList_ from './EntryList.vue'
import PathBar_ from './PathBar.vue'
import { Entry } from '@/types'

export type ListViewMode = 'list' | 'thumbnail'

export interface EntryEventData {
  entry?: Entry
  path?: string
  event?: Event
}

export type GetLinkFn = (e: Entry | string) => string | undefined

export const EntryIcon = EntryIcon_
export const EntryLink = EntryLink_
export const EntryItem = EntryItem_
export const EntryList = EntryList_
export const PathBar = PathBar_
