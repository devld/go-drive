import BaseDialog from './BaseDialog.vue'
import { IS_DEBUG } from '@/utils'
import { T } from '@/i18n'
import { createApp, defineComponent, h } from 'vue'
import dialogUse from '../dialog-use'
import { SimpleButtonType } from '@/components/SimpleButton'

export interface BaseDialogOptions {
  title?: I18nText
  confirmText?: I18nText
  confirmType?: SimpleButtonType
  cancelText?: I18nText
  cancelType?: SimpleButtonType
  transition?: string
  escClose?: boolean
  overlayClose?: boolean

  onOk?: (v?: any) => PromiseValue<any>
  onCancel?: (v?: any) => PromiseValue<any>
}

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
        escClose: false,
        overlayClose: false,

        _onClose: undefined as (() => void) | undefined,
        _callback: undefined as
          | { onOk?: (v?: any) => any; onCancel?: (v?: any) => any }
          | undefined,
        _promise: undefined as
          | { resolve: (v?: any) => any; reject: (v?: any) => any }
          | undefined,
      }
    },
    render() {
      return h(
        BaseDialog as any,
        {
          ref: 'bd',
          showing: this.showing,
          loading: this.loading,
          title: this.title,
          confirmText: this.confirmText,
          confirmType: this.confirmType,
          confirmDisabled: this.confirmDisabled,
          cancelText: this.cancelText,
          cancelType: this.cancelType,
          transition: this.transition,
          escClose: this.escClose,
          overlayClose: this.overlayClose,
          onClose: () => this.close(),
          onClosed: () => this._onClose?.(),
          onConfirm: () => this.onConfirmOrCancel(true),
          onCancel: () => this.onConfirmOrCancel(false),
        },
        {
          default: () =>
            h(component, {
              ref: 'inner',
              loading: this.loading,
              opts: this.opts,
              onLoading: (v: string | boolean) => this.toggleLoading(v),
              onConfirm: () => this.onConfirmOrCancel(true),
              onCancel: () => this.onConfirmOrCancel(false),
              onConfirmDisabled: (disabled: any) =>
                (this.confirmDisabled = !!disabled),
            }),
        }
      )
    },
    methods: {
      show(opts: BaseDialogOptions) {
        this.opts = opts

        this.title = opts.title || ''
        this.confirmText = opts.confirmText || T('dialog.base.ok')
        this.confirmType = opts.confirmType
        this.cancelText = opts.cancelText
        this.cancelType = opts.cancelType || 'info'
        this.transition = opts.transition || 'bottom-fade'
        this.escClose = !!opts.escClose
        this.overlayClose = !!opts.overlayClose

        if (
          typeof opts.onOk === 'function' ||
          typeof opts.onCancel === 'function'
        ) {
          this._callback = { onOk: opts.onOk, onCancel: opts.onCancel }
        }

        this.showing = true

        if (!this._callback) {
          return new Promise((resolve, reject) => {
            this._promise = { resolve, reject }
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

        if (this._callback) {
          t = setTimeout(() => this.toggleLoading(confirm), 0)
          try {
            if (confirm && this._callback.onOk) {
              await this._callback.onOk(val)
            }
            if (!confirm && this._callback.onCancel) {
              await this._callback.onCancel(val || 'cancel')
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

        if (this._promise) {
          if (confirm) this._promise.resolve(val)
          else this._promise.reject(val || 'cancel')
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
      close() {
        this.toggleLoading()
        this._callback && this._callback.onCancel && this._callback.onCancel()
        this._callback = undefined

        this._promise && this._promise.reject()
        this._promise = undefined

        this.showing = false
      },
      onClose(fn: () => void) {
        this._onClose = fn
      },
    },
    _base_dialog: true,
  })
}

export default function showBaseDialog<T = any>(
  component: any,
  opts: BaseDialogOptions
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
