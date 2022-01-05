import { isT, T } from '@/i18n'
import showBaseDialog, { createDialog } from '../base-dialog'
import TextDialogInner from './TextDialog.vue'

const TextDialog = createDialog('TextDialog', TextDialogInner)

export function showAlertDialog(opts) {
  if (typeof opts !== 'object' || isT(opts)) {
    opts = { message: opts }
  }
  return showBaseDialog(TextDialog, {
    ...opts,
    transition: opts.transition || 'flip-fade',
  })
}

export function showConfirmDialog(opts) {
  if (typeof opts === 'string') {
    opts = { message: opts }
  }
  return showBaseDialog(TextDialog, {
    ...opts,
    transition: opts.transition || 'flip-fade',
    confirmText: opts.confirmText || T('dialog.text.yes'),
    cancelText: opts.cancelText || T('dialog.text.no'),
  })
}
