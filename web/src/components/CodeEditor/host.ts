import {
  observeThemeChanges,
  resolveCssColor,
} from '@go-drive/utils'
import { onMounted, onUnmounted, type Ref } from 'vue'
import {
  MESSAGE_KEY_PREFIX,
  type EditorInMessageTypes,
  type MessageHandler,
} from '@go-drive/monaco-editor'

export type EditorEmit = <K extends keyof EditorInMessageTypes>(
  fn: K,
  data: EditorInMessageTypes[K]
) => void

export const useEditorSetup = (
  id: string,
  el: Ref<HTMLIFrameElement | undefined>,
  handlers: Record<string, MessageHandler>
) => {
  const messageKey = MESSAGE_KEY_PREFIX + id

  const emit: EditorEmit = (fn, data) => {
    el.value!.contentWindow!.postMessage({
      [messageKey]: [fn, data],
    })
  }

  const onMessage = (e: MessageEvent) => {
    let data = e.data
    if (
      !data ||
      typeof data !== 'object' ||
      !(data = data[messageKey]) ||
      !Array.isArray(data)
    ) {
      return
    }
    const fn = data[0]
    data = data[1]

    const handler = handlers[fn]
    if (!handler) return
    handler(data)
  }

  onMounted(() => {
    window.addEventListener('message', onMessage)
  })
  onUnmounted(() => {
    window.removeEventListener('message', onMessage)
  })

  return [emit]
}

export const useEditorTheme = (emit: EditorEmit) => {
  let stopObservingTheme: (() => void) | undefined
  const setTheme = () => {
    const root = document.documentElement
    const styles = getComputedStyle(root)
    const colorScheme = styles.colorScheme.split(/\s+/)
    const isDark = colorScheme.includes('dark')
    const background = styles.getPropertyValue('--color-bg-code-editor').trim()
    emit('setTheme', isDark ? 'vs-dark' : 'vs')
    emit('setBackground', resolveCssColor(background, root) ?? '#ffffff')
  }

  onMounted(() => {
    stopObservingTheme = observeThemeChanges(setTheme)
  })
  onUnmounted(() => {
    stopObservingTheme?.()
  })

  return [setTheme]
}
