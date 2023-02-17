import { ListViewMode } from '@/components/entry'
import {
  createViewHandler,
  executeFunctionalHandler,
  resolveEntryMenus,
} from '@/handlers'
import { useHandlerCtx } from '@/handlers/handler-ctx'
import { getHandler, resolveEntryHandler } from '@/handlers/handlers'
import {
  EntryHandler,
  EntryHandlerContext,
  EntryHandlerExecutionOption,
  EntryHandlerExecutionParams,
} from '@/handlers/types'
import { Entry, EntryPathMeta, User } from '@/types'
import { dir, getRouteQuery, pathClean } from '@/utils'
import uiUtils from '@/utils/ui-utils'
import { computed, onBeforeUnmount, ref, Ref, watch } from 'vue'
import { RouteLocationNormalizedLoaded } from 'vue-router'

export type ResolveHandlerByRoute = (
  route: RouteLocationNormalizedLoaded
) => { handler: EntryHandler; dir: string; entryName: string } | undefined

export const useEntryExplorer = (
  routeBasePath: string,
  queryHandler = 'h',
  queryEntry = 'e'
) => {
  if (!routeBasePath.endsWith('/')) routeBasePath += '/'

  const handlerCtx = useHandlerCtx()

  const resolvePath = (route: RouteLocationNormalizedLoaded) =>
    route.path.startsWith(routeBasePath)
      ? decodeURIComponent(pathClean(route.path.replace(routeBasePath, '')))
      : undefined

  const getDirLink = (path: string) => `${routeBasePath}${path}`

  const getHandlerLink = (
    dirPath: string,
    handlerName: string,
    entryName: string
  ) => {
    let path = `${routeBasePath}${dirPath}`
    if (handlerName && entryName) {
      path +=
        `?${queryHandler}=${handlerName}&` +
        `${queryEntry}=${encodeURIComponent(entryName)}`
    }
    return path
  }

  const getLink = (entry: Entry | string) => {
    if (typeof entry === 'string') return getDirLink(entry)

    if (entry.type === 'dir') return getDirLink(entry.path)
    if (entry.type === 'file') {
      const handler = resolveEntryHandler(handlerCtx.value, { entry })[0]
      if (handler && handler.view) {
        return getHandlerLink(dir(entry.path), handler.name, entry.name)
      }
    }
  }

  const resolveHandlerByRoute: ResolveHandlerByRoute = (
    route: RouteLocationNormalizedLoaded
  ) => {
    const dir = resolvePath(route)
    const handler = getHandler(getRouteQuery(route.query, queryHandler) ?? '')
    const entryName = getRouteQuery(route.query, queryEntry)
    if (dir === undefined || !handler || !entryName) return
    return { dir, handler, entryName }
  }

  const isRouteForHandlerView = (
    route: RouteLocationNormalizedLoaded,
    handler: string,
    entry: string,
    dir?: string
  ) => {
    const matched = resolveHandlerByRoute(route)
    return (
      matched &&
      (!dir || dir === matched.dir) &&
      matched.handler.name === handler &&
      matched.entryName === entry
    )
  }

  return {
    handlerCtx,
    getDirLink,
    getLink,
    getHandlerLink,
    resolveHandlerByRoute,
    isRouteForHandlerView,
    resolvePath,
  }
}

export const useEntryHandler = (
  currentDirEntry: Ref<Entry | undefined>,
  entries: Ref<Entry[] | undefined>,
  handlerCtx: Ref<EntryHandlerContext>,
  resolveHandlerByRoute: ResolveHandlerByRoute,
  onReloadEntryList: () => void,
  onHandlerExecute: (
    handler: EntryHandler,
    entry: Entry | Entry[]
  ) => PromiseValue<void>,
  onHandlerHide: (entry: Entry | Entry[]) => PromiseValue<void>,
  onEntryChange: (path: string, handlerName: string) => PromiseValue<void>
) => {
  onBeforeUnmount(() => {
    destroyViewHandler()
  })

  const { handler: viewHandler, destroy: destroyViewHandler } =
    createViewHandler()

  const getEntryHandlerData = (
    entry: Entry | Entry[]
  ): EntryHandlerExecutionParams => {
    if (!entries.value) throw new Error('not ready')
    return {
      entry,
      entries: entries.value,
      parent: currentDirEntry.value,
    }
  }

  const hideViewHandler = async () => {
    try {
      await onHandlerHide(viewHandler.data.entry)
      viewHandler.hide()
      return true
    } catch {
      // ignore
    }
    return false
  }

  const handlerOpt = computed<EntryHandlerExecutionOption>(() => ({
    ctx: handlerCtx.value,
    uiUtils,
    onRefresh: () => {
      onReloadEntryList()
    },
    onClose: () => {
      hideViewHandler()
    },
    onEntryChange: async (path: string) => {
      try {
        await onEntryChange(path, viewHandler.handler)
      } catch {
        return
      }
    },
  }))

  const getEntryMenus = (entry: Entry | Entry[]) => {
    return resolveEntryMenus(handlerCtx.value, getEntryHandlerData(entry))
  }

  const executeHandler = async (
    handlerName: string,
    entry: Entry | Entry[]
  ) => {
    const handler = getHandler(handlerName)
    if (!handler) return false

    try {
      await onHandlerExecute(handler, entry)
    } catch {
      return false
    }

    if (handler.view) {
      viewHandler.show(
        handler.name,
        getEntryHandlerData(entry),
        handlerOpt.value
      )
      return true
    } else if (handler.handler) {
      executeFunctionalHandler(
        handler.name,
        getEntryHandlerData(entry),
        handlerOpt.value
      )
      return true
    }
    return false
  }

  const onRouteChanged = async (route: RouteLocationNormalizedLoaded) => {
    const matched = resolveHandlerByRoute(route)
    if (!matched) {
      viewHandler.hide()
      return
    }

    const { handler, entryName } = matched
    const entry = entries.value?.find((e) => e.name === entryName)

    if (!entry) return false

    return executeHandler(handler.name, entry)
  }

  return {
    getViewHandlerShowing: () => viewHandler.showing,
    getViewHandlerSavedState: () => viewHandler.saved,
    hideViewHandler,
    getEntryMenus,
    executeHandler,
    onRouteChanged,
  }
}

export const useEntriesListState = (
  currentDir: Ref<Entry | undefined>,
  user: Ref<User | undefined>
) => {
  const viewMode = ref<ListViewMode>()
  const sortBy = ref<string>()
  const filterFn = ref<Fn1<Entry, boolean>>()

  const storeKey = (path: string) =>
    `entries-list:${user.value?.username || ''}:${path}`

  const init = (meta?: EntryPathMeta) => {
    const path = currentDir.value?.path
    if (typeof path !== 'string') return

    let newViewMode: ListViewMode | undefined
    let newSortBy: string | undefined
    let newFilterFn: Fn1<Entry, boolean> | undefined

    const stored = localStorage.getItem(storeKey(path))
    if (stored) {
      const segments = stored.split(',')
      newViewMode = (segments[0] as ListViewMode) || undefined
      newSortBy = segments[1] || undefined
    }

    if (meta) {
      newViewMode = newViewMode || (meta.defaultMode as ListViewMode)
      newSortBy = newSortBy || meta.defaultSort
      if (meta.hiddenPattern) {
        try {
          const re = new RegExp(meta.hiddenPattern, 'i')
          newFilterFn = (entry) => !re.test(entry.name)
        } catch {
          // ignore
        }
      }
    }

    viewMode.value = newViewMode
    sortBy.value = newSortBy
    filterFn.value = newFilterFn
  }

  const storeConfig = () => {
    const path = currentDir.value?.path
    if (typeof path !== 'string') return

    localStorage.setItem(
      storeKey(path),
      [viewMode.value, sortBy.value].join(',')
    )
  }

  watch(currentDir, (dir) => init(dir?.meta.pathMeta), { immediate: true })

  const onViewModeChange = (newViewMode: ListViewMode) => {
    viewMode.value = newViewMode
    storeConfig()
  }

  const onSortByChange = (newSortBy: string) => {
    sortBy.value = newSortBy
    storeConfig()
  }

  return { viewMode, sortBy, filterFn, onViewModeChange, onSortByChange }
}
