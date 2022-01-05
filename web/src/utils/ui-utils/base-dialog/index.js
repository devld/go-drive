import BaseDialog from './BaseDialog.vue'
import { IS_DEBUG } from '@/utils'
import { T } from '@/i18n'
import { createApp, h } from 'vue'
import dialogUse from '../dialog-use'

/**
 * @typedef Options
 * @property {string} title dialog title
 * @property {string} confirmText confirm button text
 * @property {Function} onOk called when user confirmed, optional return an rejected Promise to prevent dialog from closing
 * @property {Function} onCancel called when user canceled
 */

/**
 * create a dialog wrapper of the given component
 * @param {string} name
 * @param {import('vue').Component} component
 * @returns {import('vue').Component}
 */
export function createDialog(name, component) {
  return {
    name,
    data() {
      return {
        loading: '',
        opts: {},

        title: '',
        confirmText: '',
        confirmType: '',
        confirmDisabled: false,
        cancelText: '',
        cancelType: '',
        showing: false,

        transition: '',
        escClose: false,
        overlayClose: false,
      }
    },
    render() {
      return h(
        BaseDialog,
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
              onLoading: (v) => this.toggleLoading(v),
              onConfirm: () => this.onConfirmOrCancel(true),
              onCancel: () => this.onConfirmOrCancel(false),
              onConfirmDisabled: (disabled) =>
                (this.confirmDisabled = !!disabled),
            }),
        }
      )
    },
    methods: {
      show(opts) {
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
      beforeConfirmOrCancel(confirm) {
        const inner = this.$refs.inner
        if (!inner) return
        let cb
        if (confirm && (cb = inner.beforeConfirm)) return cb && cb()
        if (!confirm && (cb = inner.beforeCancel)) return cb && cb()
      },
      async onConfirmOrCancel(confirm) {
        let val
        let t = setTimeout(() => this.toggleLoading(confirm), 0)
        try {
          val = await this.beforeConfirmOrCancel(confirm)
        } catch (e) {
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
          } catch (e) {
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
      toggleLoading(loading) {
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
        this._callback = null

        this._promise && this._promise.reject()
        this._promise = null

        this.showing = false
      },
      onClose(fn) {
        this._onClose = fn
      },
    },
    _base_dialog: true,
  }
}

export default function showBaseDialog(component, opts) {
  if (!component._base_dialog) throw new Error()

  const div = document.createElement('div')
  document.body.appendChild(div)

  const vm = createApp(component).use(dialogUse).mount(div)
  vm.onClose(() => {
    vm.$.appContext.app.unmount()
    document.body.removeChild(div)
  })

  return vm.show(opts)
}
