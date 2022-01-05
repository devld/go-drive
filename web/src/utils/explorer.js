import { resolveEntryHandler } from '@/utils/handlers'
import { dir, pathClean } from '.'
import { getHandler } from './handlers'

/**
 * @param {string} routeBasePath
 */
export const useEntryExplorer = (
  routeBasePath,
  queryHandler = 'h',
  queryEntry = 'e'
) => {
  const getDirLink = (path) => `${routeBasePath}/${path}`

  const getHandlerLink = (handlerName, entryName, dirPath) => {
    return (
      `${routeBasePath}/${dirPath}?${queryHandler}=${handlerName}&` +
      `${queryEntry}=${encodeURIComponent(entryName)}`
    )
  }

  const getLink = (entry) => {
    if (typeof entry === 'string') return getDirLink(entry)

    if (entry.type === 'dir') return getDirLink(entry.path)
    if (entry.type === 'file') {
      const handler = resolveEntryHandler(entry)[0]
      if (handler && handler.view) {
        return getHandlerLink(handler.name, entry.name, dir(entry.path))
      }
    }
  }

  const resolveHandlerByRoute = (route) => {
    const handler = getHandler(route.query[queryHandler])
    const entryName = route.query[queryEntry]
    if (!handler || !entryName) {
      return null
    }
    return { handler, entryName }
  }

  const resolvePath = (route) =>
    pathClean(route.path.replace(routeBasePath, ''))

  return {
    getDirLink,
    getLink,
    getHandlerLink,
    resolveHandlerByRoute,
    resolvePath,
  }
}
