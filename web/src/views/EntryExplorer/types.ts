import { EntryHandlerMenuItem } from '@/handlers/types'
import type { Entry } from '@/types'

export interface EntryMenuClickData {
  entry: Entry | Entry[]
  menu: EntryHandlerMenuItem
}
