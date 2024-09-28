import { Entry } from '@/types'
import { mapOf } from '@/utils'
import {
  EntryHandler,
  EntryHandlerContext,
  EntryHandlerExecutionParams,
  EntryHandlerSupportsParams,
} from './types'

export const HANDLERS: Readonly<EntryHandler[]> = Object.freeze(
  (
    Object.values(import.meta.glob('./*/index.ts', { eager: true })) as Record<
      string,
      EntryHandler
    >[]
  )
    .flatMap((m) => Object.values(m))
    .map((e) => ({
      handler: e,
      typeOrder: e.view ? -1 : 1,
      order: e.order ?? 0,
    }))
    .sort((a, b) => a.typeOrder - b.typeOrder || a.order - b.order)
    .map((e) => e.handler)
)

export const HANDLER_COMPONENTS = mapOf(
  HANDLERS.filter((h) => h.view),
  (h) => h.view!.name,
  (h) => h.view!.component
)

const HANDLERS_MAP = mapOf(HANDLERS, (h) => h.name)

export function getHandler(name: string): EntryHandler | undefined {
  return HANDLERS_MAP[name]
}

export function processEntryHandlerExecutionParams(
  data: EntryHandlerExecutionParams,
  handler: EntryHandler
): EntryHandlerExecutionParams {
  const entries = Array.isArray(data.entry) ? data.entry : [data.entry]
  if (entries.length === 0) throw new Error('empty data')
  return {
    ...data,
    entry: handler.multiple ? entries : entries[0],
  }
}

export function isHandlerSupports(
  handler: EntryHandler,
  ctx: EntryHandlerContext,
  data: EntryHandlerSupportsParams<Entry | Entry[]>
) {
  const entry = data.entry
  const entries = Array.isArray(entry) ? entry : [entry]
  if (entries.length === 0) return false
  if (!handler.multiple && entries.length > 1) return false
  if (handler.multiple) {
    return handler.supports({ entry: entries, parent: data.parent }, ctx)
  } else {
    return handler.supports({ entry: entries[0], parent: data.parent }, ctx)
  }
}

export function resolveEntryHandler(
  ctx: EntryHandlerContext,
  data: EntryHandlerSupportsParams<Entry | Entry[]>
) {
  return HANDLERS.filter((h) => isHandlerSupports(h, ctx, data))
}
