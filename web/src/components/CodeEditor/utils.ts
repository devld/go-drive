import {
  addPreferColorListener,
  isDarkMode,
  removePreferColorListener,
} from '@/utils/theme'
import { onMounted, onUnmounted, Ref } from 'vue'
import {
  MESSAGE_KEY_PREFIX,
  MessageHandler,
  EditorInMessageTypes,
} from '../../../monaco-editor/src/types'

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
  const setTheme = () => {
    emit('setTheme', isDarkMode() ? 'vs-dark' : 'vs')
  }

  addPreferColorListener(setTheme)
  onUnmounted(() => {
    removePreferColorListener(setTheme)
  })

  return [setTheme]
}
