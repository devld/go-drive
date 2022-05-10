import { mapOf } from '@/utils'
import deleteEntry from './delete'
import download from './download'
import image from './image'
import media from './media'
import permission from './permission'
import textEdit from './text-edit'
import { copy, move } from './copy-move'
import rename from './rename'
import mount from './mount'

export const HANDLERS = Object.freeze([
  textEdit,
  image,
  media,
  download,
  deleteEntry,
  rename,
  copy,
  move,
  permission,
  mount,
])

export const HANDLER_COMPONENTS = mapOf(
  HANDLERS.filter((h) => h.view),
  (h) => h.view.name,
  (h) => h.view.component
)

const HANDLERS_MAP = mapOf(HANDLERS, (h) => h.name)

export function getHandler(name) {
  return HANDLERS_MAP[name] && HANDLERS_MAP[name]
}

export function resolveEntryHandler(entry, parentEntry, ctx) {
  const matches = []
  const isMultiple = Array.isArray(entry)
  for (const h of HANDLERS) {
    if (isMultiple && !h.multiple) continue
    if (h.supports(entry, parentEntry, ctx)) matches.push(h)
  }
  return matches
}
