import {
  showAlertDialog,
  showConfirmDialog,
  TextDialogOptions,
} from './text-dialog'
import showInputDialog, { InputDialogOptions } from './input-dialog'
import toggleLoadingDialog, { LoadingOptions } from './loading-dialog'
import showOpenDialog, { OpenDialogOptions } from './open-dialog'
import showBaseDialog, { BaseDialogOptions } from './base-dialog'

export const dialog = <
  OT extends BaseDialogOptions = BaseDialogOptions,
  RT = any
>(
  component: any,
  opts: OT
) => showBaseDialog<RT>(component, opts)

export const alert = (opts: TextDialogOptions | I18nText) =>
  showAlertDialog(opts)
export const confirm = (opts: TextDialogOptions | I18nText) =>
  showConfirmDialog(opts)
export const input = (opts: InputDialogOptions) => showInputDialog(opts)
export const loading = (opts?: LoadingOptions | boolean) =>
  toggleLoadingDialog(opts)
export const open = (opts: OpenDialogOptions) => showOpenDialog(opts)

export interface UIUtils {
  dialog: typeof dialog
  open: typeof open
  alert: typeof alert
  confirm: typeof confirm
  input: typeof input
  loading: typeof loading
}

export default {
  dialog,
  open,
  alert,
  confirm,
  input,
  loading,
} as UIUtils
