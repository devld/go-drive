import TextDialogInner from './TextDialog.vue'
import showBaseDialog, { createDialog } from '../base-dialog'

const TextDialog = createDialog('TextDialog', TextDialogInner)

export function showAlertDialog (Vue, opts) {
  if (typeof (opts) !== 'object') {
    opts = { message: opts }
  }
  return showBaseDialog(Vue, TextDialog, { ...opts, transition: opts.transition || 'flip-fade' })
}

export function showConfirmDialog (Vue, opts) {
  if (typeof (opts) === 'string') {
    opts = { message: opts }
  }
  return showBaseDialog(Vue, TextDialog, {
    ...opts,
    transition: opts.transition || 'flip-fade',
    confirmText: opts.confirmText || 'Yes',
    cancelText: opts.cancelText || 'No'
  })
}
