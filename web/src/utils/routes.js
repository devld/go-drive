import { resolveEntryHandler } from '@/utils/handlers'
import { dir } from '.'

export const BASE_PATH = '/files'
export const QUERY_HANDLER = 'handler'
export const QUERY_ENTRY = 'entry'

export function getBaseLink (path) {
  return `${BASE_PATH}/${path}`
}

export function getDirEntryLink (path) {
  return `${BASE_PATH}/${path}`
}

export function makeEntryLink (entry) {
  if (entry.type === 'dir') return getDirEntryLink(entry.path)
  if (entry.type === 'file') {
    const handler = resolveEntryHandler(entry)[0]
    if (handler && handler.view) {
      return makeEntryHandlerLink(handler.name, entry.name, dir(entry.path))
    }
  }
}

export function makeEntryHandlerLink (handlerName, entryName, dirPath) {
  return `${BASE_PATH}/${dirPath}?${QUERY_HANDLER}=${handlerName}&` +
    `${QUERY_ENTRY}=${encodeURIComponent(entryName)}`
}
