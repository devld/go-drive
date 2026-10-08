import { autoUpdate } from '@floating-ui/dom'
import { editorViewCtx } from '@milkdown/kit/core'
import { BlockService } from '@milkdown/kit/plugin/block'

/** Keep Crepe's handle anchored when scrolling or lazy code blocks change layout. */
export function createBlockService() {
  const service = new BlockService()
  const bind = service.bind
  const unBind = service.unBind
  let stopAutoUpdate: (() => void) | undefined

  service.bind = (ctx, notify) => {
    stopAutoUpdate?.()
    bind(ctx, (message) => {
      stopAutoUpdate?.()
      stopAutoUpdate = undefined
      if (message.type === 'hide') {
        notify(message)
        return
      }

      const editor = ctx.get(editorViewCtx).dom
      const handle = editor.parentElement?.querySelector<HTMLElement>(
        '.milkdown-block-handle'
      )
      if (!handle) {
        notify(message)
        return
      }

      // Reposition the same active block, without selecting another block at
      // the pointer's old screen coordinates after the document has shifted.
      stopAutoUpdate = autoUpdate(message.active.el, handle, () => {
        const block = message.active.el
        const viewport = editor
          .closest('.markdown-wysiwyg-editor')
          ?.getBoundingClientRect()
        const rect = block.getBoundingClientRect()
        if (
          !block.isConnected ||
          (viewport && (rect.bottom <= viewport.top || rect.top >= viewport.bottom))
        ) {
          notify({ type: 'hide' })
          return
        }
        notify(message)
      })
    })
  }

  service.unBind = () => {
    stopAutoUpdate?.()
    stopAutoUpdate = undefined
    unBind()
  }

  return service
}
