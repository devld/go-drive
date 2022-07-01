import { isT, T } from '@/i18n'
import showBaseDialog, { BaseDialogOptions, createDialog } from '../base-dialog'
import TextDialogInner from './TextDialog.vue'

export interface TextDialogOptions extends BaseDialogOptions {
  message?: I18nText
}

const TextDialog = createDialog('TextDialog', TextDialogInner)

export function showAlertDialog(opts: TextDialogOptions | I18nText) {
  if (typeof opts !== 'object' || isT(opts)) {
    opts = { message: opts }
  }
  return showBaseDialog<void>(TextDialog, {
    ...opts,
    transition: opts.transition || 'flip-fade',
  })
}

export function showConfirmDialog(opts: TextDialogOptions | I18nText) {
  if (typeof opts !== 'object' || isT(opts)) {
    opts = { message: opts }
  }
  return showBaseDialog<void>(TextDialog, {
    ...opts,
    transition: opts.transition || 'flip-fade',
    confirmText: opts.confirmText || T('dialog.text.yes'),
    cancelText: opts.cancelText || T('dialog.text.no'),
  })
}
