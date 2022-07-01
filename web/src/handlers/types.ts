import { Config, Entry, User } from '@/types'
import { UIUtils } from '@/utils/ui-utils'

export interface EntryHandlerExecutionOption {
  ctx: EntryHandlerContext
  uiUtils: UIUtils
  onRefresh?: () => void
  onClose?: (entry: Entry | Entry[]) => void
  onEntryChange?: (path: string) => void
}

export interface EntryHandlerViewHandle {
  show: (
    handlerName: string,
    data: EntryHandlerExecutionParams,
    opt: EntryHandlerExecutionOption
  ) => false | undefined
  hide: () => void

  showing: boolean
  saved: boolean
  handler: string
  data: EntryHandlerExecutionParams
}

export interface EntryHandlerMenuItem {
  /** handler name */
  name: string
  display: EntryHandlerDisplayConfig
}

export interface EntryHandlersMenu {
  entry: Entry | Entry[]
  menus: EntryHandlerMenuItem[]
}

export interface EntryHandlerDisplayConfig {
  name: I18nText
  description?: I18nText
  icon?: string
  type?: 'danger'
}

export type EntryHandlerDisplay<A> =
  | EntryHandlerDisplayConfig
  | ((e: A) => EntryHandlerDisplayConfig)

export interface EntryHandlerView {
  name: string
  component: any
}

export interface EntryHandlerContext {
  user?: User
  config: Config
  options: O<string>
}

export interface EntryHandlerSupportsParams<A = Entry | Entry[]> {
  entry: A
  parent?: Entry
}

export interface EntryHandlerExecutionParams<A = Entry | Entry[]>
  extends EntryHandlerSupportsParams<A> {
  entries: Entry[]
}

export type EntrySupportsFunc<A> = (
  data: EntryHandlerSupportsParams<A>,
  ctx: EntryHandlerContext
) => boolean

export interface EntryHandlerFuncReturns {
  /** should the entries list update */
  update?: boolean
}

export type EntryHandlerFunc<A> = (
  data: EntryHandlerExecutionParams<A>,
  uiUtils: UIUtils,
  ctx: EntryHandlerContext
) => Promise<EntryHandlerFuncReturns | undefined>

export interface BaseEntryHandler<A> {
  name: string
  display: EntryHandlerDisplay<A>
  supports: EntrySupportsFunc<A>
  multiple?: boolean
  view?: EntryHandlerView
  handler?: EntryHandlerFunc<A>

  /** resolving order */
  order?: number
}

export interface SingleEntryHandler extends BaseEntryHandler<Entry> {
  multiple?: false
}
export interface MultipleEntriesHandler extends BaseEntryHandler<Entry[]> {
  multiple: true
}

export type EntryHandler = SingleEntryHandler | MultipleEntriesHandler
