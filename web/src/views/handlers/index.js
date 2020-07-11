import { mapOf, cloneObject } from '@/utils'

import TextEditView from './TextEditView'
import ImageView from './ImageView'
import DownloadView from './DownloadView'

export const HANDLERS = Object.freeze([
  TextEditView, ImageView, DownloadView
])

export const HANDLER_COMPONENTS = mapOf(HANDLERS.filter(h => h.view), h => h.view.name, h => h.view.component)

const HANDLERS_MAP = mapOf(HANDLERS, h => h.name)

export function getHandler (name) {
  return HANDLERS_MAP[name] && cloneObject(HANDLERS_MAP[name])
}

export function resolveEntryHandler (entry) {
  const matches = []
  for (const h of HANDLERS) {
    if (h.supports(entry)) matches.push(cloneObject(h))
  }
  return matches
}

window.resolveEntryHandler = resolveEntryHandler
