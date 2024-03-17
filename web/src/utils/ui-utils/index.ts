import showBaseDialog, { BaseDialogOptions } from './base-dialog'
import showInputDialog from './input-dialog'
import toggleLoadingDialog from './loading-dialog'
import showOpenDialog from './open-dialog'
import { showAlertDialog, showConfirmDialog } from './text-dialog'

export const dialog = <
  OT extends BaseDialogOptions = BaseDialogOptions,
  RT = any
>(
  component: any,
  opts: OT
) => showBaseDialog<RT>(component, opts)

export const alert = showAlertDialog
export const confirm = showConfirmDialog
export const input = showInputDialog
export const loading = toggleLoadingDialog
export const open = showOpenDialog

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
