import { cloneObject, mapOf } from '@/utils'
import deleteEntry from './delete'
import download from './download'
import image from './image'
import permission from './permission'
import textEdit from './text-edit'

export const HANDLERS = Object.freeze([
  textEdit, image, download, deleteEntry,
  permission
])

export const HANDLER_COMPONENTS = mapOf(HANDLERS.filter(h => h.view), h => h.view.name, h => h.view.component)

const HANDLERS_MAP = mapOf(HANDLERS, h => h.name)

export function getHandler (name) {
  return HANDLERS_MAP[name] && cloneObject(HANDLERS_MAP[name])
}

export function resolveEntryHandler (entry, user) {
  const matches = []
  for (const h of HANDLERS) {
    if (h.supports(entry, user)) matches.push(cloneObject(h))
  }
  return matches
}
