import BaseDialog from './BaseDialog.vue'
import { IS_DEBUG } from '@/utils'
import { T } from '@/i18n'
import { createApp, defineComponent, h } from 'vue'
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
]

export function createDialog(name: string, component: any) {
  return defineComponent({
    name,
    data() {
      return {
        loading: '',
        opts: {} as BaseDialogOptions,

        title: '' as I18nText,
        confirmText: '' as I18nText | undefined,
        confirmType: '' as SimpleButtonType | undefined,
        confirmDisabled: false,
        cancelText: '' as I18nText | undefined,
        cancelType: '' as SimpleButtonType | undefined,
        showing: false,

        transition: '',
        closeable: true,
        escClose: false,
        overlayClose: false,

        onClose_: undefined as (() => void) | undefined,
        callback_: undefined as
          | { onOk?: (v?: any) => any; onCancel?: (v?: any) => any }
          | undefined,
        promise_: undefined as
          | { resolve: (v?: any) => any; reject: (v?: any) => any }
          | undefined,
      }
    },
    methods: {
      show(opts: BaseDialogOptions) {
        this.opts = opts

        this.title = opts.title || ''
        this.confirmText = opts.confirmText
        this.confirmType = opts.confirmType
        this.cancelText = opts.cancelText
        this.cancelType = opts.cancelType || 'info'
        this.transition = opts.transition || 'bottom-fade'
        this.closeable = opts.closeable ?? true
        this.escClose = !!opts.escClose
        this.overlayClose = !!opts.overlayClose

        if (
          typeof opts.onOk === 'function' ||
          typeof opts.onCancel === 'function'
        ) {
          this.callback_ = { onOk: opts.onOk, onCancel: opts.onCancel }
        }

        this.showing = true

        if (!this.callback_) {
          return new Promise((resolve, reject) => {
            this.promise_ = { resolve, reject }
          })
        }
      },
      beforeConfirmOrCancel(confirm: boolean) {
        const inner = this.$refs.inner as any
        if (!inner) return
        let cb
        if (confirm && (cb = inner.beforeConfirm)) return cb && cb()
        if (!confirm && (cb = inner.beforeCancel)) return cb && cb()
      },
      async onConfirmOrCancel(confirm: boolean) {
        let val
        let t = setTimeout(() => this.toggleLoading(confirm), 0)
        try {
          val = await this.beforeConfirmOrCancel(confirm)
        } catch (e: any) {
          if (IS_DEBUG) {
            console.warn(e)
          }
          return
        } finally {
          clearTimeout(t)
          this.toggleLoading()
        }

        if (this.callback_) {
          t = setTimeout(() => this.toggleLoading(confirm), 0)
          try {
            if (confirm && this.callback_.onOk) {
              await this.callback_.onOk(val)
            }
            if (!confirm && this.callback_.onCancel) {
              await this.callback_.onCancel(val || 'cancel')
            }
          } catch (e: any) {
            console.warn('dialog callback error', e)
            return
          } finally {
            clearTimeout(t)
            this.toggleLoading()
          }
          this.close()
        }

        if (this.promise_) {
          if (confirm) this.promise_.resolve(val)
          else this.promise_.reject(val || 'cancel')
          this.close()
        }
      },
      toggleLoading(loading?: string | boolean) {
        if (typeof loading !== 'string' && typeof loading !== 'boolean') {
          this.loading = ''
          return
        }
        if (loading === 'confirm' || loading === 'cancel' || loading === '') {
          this.loading = loading
          return
        }
        this.loading = loading ? 'confirm' : 'cancel'
      },
      onOptionsChange(opts: Partial<BaseDialogOptionsData>) {
        Object.keys(opts).forEach((key) => {
          if (!ON_OPTIONS_KEYS.includes(key)) return
          ;(this.$data as any)[key] = opts[
            key as keyof BaseDialogOptionsData
          ] as any
        })
      },
      close() {
        this.toggleLoading()
        this.callback_ && this.callback_.onCancel && this.callback_.onCancel()
        this.callback_ = undefined

        this.promise_ && this.promise_.reject()
        this.promise_ = undefined

        this.showing = false
      },
      onClose(fn: () => void) {
        this.onClose_ = fn
      },
    },
    render() {
      return h(
        BaseDialog as any,
        {
          ref: 'bd',
          showing: this.showing,
          loading: this.loading,
          title: this.title,
          confirmText: this.confirmText || T('dialog.base.ok'),
          confirmType: this.confirmType,
          confirmDisabled: this.confirmDisabled,
          cancelText: this.cancelText,
          cancelType: this.cancelType,
          transition: this.transition,
          escClose: this.escClose,
          closeable: this.closeable,
          overlayClose: this.overlayClose,
          onClose: () => this.close(),
          onClosed: () => this.onClose_?.(),
          onConfirm: () => this.onConfirmOrCancel(true),
          onCancel: () => this.onConfirmOrCancel(false),
        },
        {
          default: () =>
            h(component, {
              ref: 'inner',
              loading: this.loading,
              opts: this.opts,
              onLoading: this.toggleLoading,
              onConfirm: () => this.onConfirmOrCancel(true),
              onCancel: () => this.onConfirmOrCancel(false),
              onOptions: this.onOptionsChange,
            }),
        }
      )
    },
    _base_dialog: true,
  })
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
