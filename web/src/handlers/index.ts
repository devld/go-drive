import Components from '@/components'
import i18n from '@/i18n'
import store from '@/store'
import Utils from '@/utils'
import { App, ComponentPublicInstance, createApp } from 'vue'
import {
  getHandler,
  isHandlerSupports,
  processEntryHandlerExecutionParams,
  resolveEntryHandler,
} from './handlers'
import HandlerViewDialog from './HandlerViewDialog.vue'
import {
  EntryHandlerContext,
  EntryHandlerExecutionOption,
  EntryHandlerExecutionParams,
  EntryHandlersMenu,
  EntryHandlerViewHandle,
} from './types'

export { getHandler } from './handlers'

export function resolveEntryMenus(
  ctx: EntryHandlerContext,
  data: EntryHandlerExecutionParams
): EntryHandlersMenu | undefined {
  const handlers = resolveEntryHandler(ctx, data)
  if (handlers.length === 0) return undefined

  return {
    entry: data.entry,
    menus: handlers
      .filter((h) => h.display)
      .map((h) => ({
        name: h.name,
        display:
          typeof h.display === 'function'
            ? h.display(data.entry as any)
            : h.display,
      })),
  }
}

export const createViewHandler = () => {
  let handlerRootEl: HTMLElement | undefined = document.createElement('div')
  document.body.appendChild(handlerRootEl)
  handlerRootEl.classList.add('handler-view-container')

  let app: App | undefined = createApp(HandlerViewDialog)
  app.use(Utils).use(Components).use(store).use(i18n)

  let handlerRootInstance: ComponentPublicInstance | undefined =
    app.mount(handlerRootEl)
  let dialog: EntryHandlerViewHandle | undefined =
    handlerRootInstance as unknown as EntryHandlerViewHandle

  const showHandlerView = (
    handlerName: string,
    data: EntryHandlerExecutionParams,
    opt: EntryHandlerExecutionOption
  ) => {
    const handler = getHandler(handlerName)
    if (!handler) return false

    return dialog!.show(
      handlerName,
      processEntryHandlerExecutionParams(data, handler),
      opt
    )
  }

  const destroy = () => {
    if (!handlerRootEl) throw new Error('destroyed')
    dialog!.hide()
    app!.unmount()
    document.body.removeChild(handlerRootEl)

    handlerRootEl = undefined
    handlerRootInstance = undefined
    app = undefined
    dialog = undefined
  }

  return {
    handler: new Proxy(dialog, {
      get(t, p, r) {
        if (!handlerRootEl) throw new Error('destroyed')
        if (p === 'show') return showHandlerView
        return Reflect.get(t, p, r)
      },
    }),
    destroy,
  }
}

export const executeFunctionalHandler = (
  handlerName: string,
  data: EntryHandlerExecutionParams,
  opt: EntryHandlerExecutionOption
) => {
  const h = getHandler(handlerName)
  if (!h || !isHandlerSupports(h, opt.ctx, data)) return false

  if (typeof h.handler !== 'function') return false

  h.handler(
    processEntryHandlerExecutionParams(data, h) as any,
    opt.uiUtils,
    opt.ctx
  ).then(
    (r) => {
      if (r?.update) opt.onRefresh?.()
    },
    (e) => {
      console.error('entry handler error', e)
    }
  )
}
