import { mapOf } from '@/utils'
import deleteEntry from './delete'
import download from './download'
import image from './image'
import permission from './permission'
import textEdit from './text-edit'
import { copy, move } from './copy-move'
import rename from './rename'

export const HANDLERS = Object.freeze([
  textEdit, image, download, deleteEntry,
  rename, copy, move, permission
])

export const HANDLER_COMPONENTS = mapOf(HANDLERS.filter(h => h.view), h => h.view.name, h => h.view.component)

const HANDLERS_MAP = mapOf(HANDLERS, h => h.name)

export function getHandler (name) {
  return HANDLERS_MAP[name] && HANDLERS_MAP[name]
}

export function resolveEntryHandler (entry, user) {
  const matches = []
  const isMultiple = Array.isArray(entry)
  for (const h of HANDLERS) {
    if (isMultiple && !h.multiple) continue
    if (h.supports(entry, user)) matches.push(h)
  }
  return matches
}
