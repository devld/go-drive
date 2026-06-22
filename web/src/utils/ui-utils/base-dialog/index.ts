import BaseDialog from './BaseDialog.vue'
import { IS_DEBUG } from '@/utils'
import { T } from '@/i18n'
import { createApp, defineComponent, h, reactive, ref, shallowRef } from 'vue'
import dialogUse from '../dialog-use'
import { SimpleButtonType } from '@/components/SimpleButton'

export interface BaseDialogOptions<OKV = any, CV = any> {
  title?: I18nText
  confirmText?: I18nText
  confirmType?: SimpleButtonType
  cancelText?: I18nText
  cancelType?: SimpleButtonType
  transition?: string
  closeable?: boolean
  escClose?: boolean
  overlayClose?: boolean

  onOk?: (v: OKV) => PromiseValue<any>
  onCancel?: (v: CV) => PromiseValue<any>
}

export interface BaseDialogOptionsData {
  title: I18nText
  confirmText: I18nText
  confirmType: SimpleButtonType
  confirmDisabled: boolean
  cancelText: I18nText
  cancelType: SimpleButtonType
  escClose: boolean
  overlayClose: boolean
}

const ON_OPTIONS_KEYS = [
  'title',
  'confirmText',
  'confirmType',
  'confirmDisabled',
  'cancelText',
  'cancelType',
  'escClose',
  'overlayClose',
] as const

type LoadingState = '' | 'confirm' | 'cancel'

/** Inner dialog component exposing optional confirm/cancel guards. */
interface DialogInner {
  beforeConfirm?: () => PromiseValue<any>
  beforeCancel?: () => PromiseValue<any>
}

/** Manages the loading indicator and deferring it to avoid flicker. */
function useLoading() {
  const loading = ref<LoadingState>('')

  const toggleLoading = (value?: string | boolean) => {
    if (typeof value !== 'string' && typeof value !== 'boolean') {
      loading.value = ''
      return
    }
    if (value === 'confirm' || value === 'cancel' || value === '') {
      loading.value = value
      return
    }
    loading.value = value ? 'confirm' : 'cancel'
  }

  // Only show the loading indicator if `task` does not settle synchronously,
  // which avoids a flicker for fast operations.
  const runWithLoading = async <T>(
    confirm: boolean,
    task: () => PromiseValue<T>
  ): Promise<T> => {
    const timer = setTimeout(() => toggleLoading(confirm), 0)
    try {
      return await task()
    } finally {
      clearTimeout(timer)
      toggleLoading()
    }
  }

  return { loading, toggleLoading, runWithLoading }
}

interface DialogDisplayState {
  title: I18nText
  confirmText?: I18nText
  confirmType?: SimpleButtonType
  confirmDisabled: boolean
  cancelText?: I18nText
  cancelType?: SimpleButtonType
  transition: string
  closeable: boolean
  escClose: boolean
  overlayClose: boolean
}

/** Reactive display options forwarded to BaseDialog. */
function useDialogOptions() {
  const state = reactive<DialogDisplayState>({
    title: '',
    confirmText: undefined,
    confirmType: undefined,
    confirmDisabled: false,
    cancelText: undefined,
    cancelType: 'info',
    transition: '',
    closeable: true,
    escClose: true,
    overlayClose: false,
  })

  const applyOptions = (opts: BaseDialogOptions) => {
    state.title = opts.title || ''
    state.confirmText = opts.confirmText
    state.confirmType = opts.confirmType
    state.cancelText = opts.cancelText
    state.cancelType = opts.cancelType || 'info'
    state.transition = opts.transition || 'bottom-fade'
    state.closeable = opts.closeable ?? true
    state.escClose = opts.escClose ?? state.closeable
    state.overlayClose = !!opts.overlayClose
  }

  const updateOptions = (opts: Partial<BaseDialogOptionsData>) => {
    ;(Object.keys(opts) as (keyof BaseDialogOptionsData)[]).forEach((key) => {
      if (!ON_OPTIONS_KEYS.includes(key as (typeof ON_OPTIONS_KEYS)[number])) {
        return
      }
      ;(state as any)[key] = opts[key]
    })
  }

  return { state, applyOptions, updateOptions }
}

/**
 * Bridges the dialog result to either callbacks (onOk/onCancel) or a Promise,
 * depending on how the dialog was opened.
 */
function useDialogResult() {
  let callbacks:
    | { onOk?: (v?: any) => any; onCancel?: (v?: any) => any }
    | undefined
  let promise:
    | { resolve: (v?: any) => void; reject: (v?: any) => void }
    | undefined

  /** Initialize for a new dialog; returns a Promise unless callbacks are used. */
  const setup = (opts: BaseDialogOptions): Promise<any> | undefined => {
    callbacks = undefined
    promise = undefined
    if (
      typeof opts.onOk === 'function' ||
      typeof opts.onCancel === 'function'
    ) {
      callbacks = { onOk: opts.onOk, onCancel: opts.onCancel }
      return
    }
    return new Promise((resolve, reject) => {
      promise = { resolve, reject }
    })
  }

  const hasCallbacks = () => !!callbacks

  const runCallback = async (confirm: boolean, val: any) => {
    if (!callbacks) return
    if (confirm) await callbacks.onOk?.(val)
    else await callbacks.onCancel?.(val || 'cancel')
  }

  const clear = () => {
    callbacks = undefined
    promise = undefined
  }

  /** Resolve the pending Promise (callbacks already ran) and stop tracking. */
  const settle = (confirm: boolean, val: any) => {
    if (promise) {
      if (confirm) promise.resolve(val)
      else promise.reject(val || 'cancel')
    }
    clear()
  }

  /** External dismiss (overlay/esc/close button) is treated as a cancel. */
  const dismiss = () => {
    callbacks?.onCancel?.()
    promise?.reject()
    clear()
  }

  return { setup, hasCallbacks, runCallback, settle, dismiss }
}

export function createDialog(name: string, component: any) {
  const Dialog = defineComponent({
    name,
    setup(_, { expose }) {
      const showing = ref(false)
      const opts = shallowRef<BaseDialogOptions>({})
      const inner = ref<DialogInner>()

      const { loading, toggleLoading, runWithLoading } = useLoading()
      const options = useDialogOptions()
      const result = useDialogResult()

      let onCloseCb: (() => void) | undefined

      const show = (o: BaseDialogOptions) => {
        opts.value = o
        options.applyOptions(o)
        const promise = result.setup(o)
        showing.value = true
        return promise
      }

      const hide = () => {
        toggleLoading()
        showing.value = false
      }

      // Triggered by overlay click / esc / close button.
      const dismiss = () => {
        result.dismiss()
        hide()
      }

      const runInnerGuard = (confirm: boolean) => {
        const target = inner.value
        if (!target) return
        const guard = confirm ? target.beforeConfirm : target.beforeCancel
        return guard?.()
      }

      const confirmOrCancel = async (confirm: boolean) => {
        let val
        try {
          val = await runWithLoading(confirm, () => runInnerGuard(confirm))
        } catch (e) {
          // A rejected guard (e.g. failed validation) keeps the dialog open.
          if (IS_DEBUG) console.warn(e)
          return
        }

        if (result.hasCallbacks()) {
          try {
            await runWithLoading(confirm, () =>
              result.runCallback(confirm, val)
            )
          } catch (e) {
            console.warn('dialog callback error', e)
            return
          }
        }

        result.settle(confirm, val)
        hide()
      }

      const onClose = (fn: () => void) => {
        onCloseCb = fn
      }

      expose({ show, onClose })

      return () =>
        h(
          BaseDialog as any,
          {
            showing: showing.value,
            loading: loading.value,
            title: options.state.title,
            confirmText: options.state.confirmText || T('dialog.base.ok'),
            confirmType: options.state.confirmType,
            confirmDisabled: options.state.confirmDisabled,
            cancelText: options.state.cancelText,
            cancelType: options.state.cancelType,
            transition: options.state.transition,
            escClose: options.state.escClose,
            closeable: options.state.closeable,
            overlayClose: options.state.overlayClose,
            onClose: dismiss,
            onClosed: () => onCloseCb?.(),
            onConfirm: () => confirmOrCancel(true),
            onCancel: () => confirmOrCancel(false),
          },
          {
            default: () =>
              h(component, {
                ref: inner,
                loading: loading.value,
                opts: opts.value,
                onLoading: toggleLoading,
                onConfirm: () => confirmOrCancel(true),
                onCancel: () => confirmOrCancel(false),
                onOptions: options.updateOptions,
              }),
          }
        )
    },
  })

  ;(Dialog as any)._base_dialog = true
  return Dialog
}

export default function showBaseDialog<T = any>(
  component: any,
  opts: BaseDialogOptions<T>
): Promise<T> {
  if (!component._base_dialog) throw new Error()

  const div = document.createElement('div')
  document.body.appendChild(div)

  const vm = createApp(component).use(dialogUse).mount(div) as any
  vm.onClose(() => {
    vm.$.appContext.app.unmount()
    document.body.removeChild(div)
  })

  return vm.show(opts)
}
