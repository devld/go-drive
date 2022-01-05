import { showAlertDialog, showConfirmDialog } from './text-dialog'
import showInputDialog from './input-dialog'
import toggleLoadingDialog from './loading-dialog'
import showOpenDialog from './open-dialog'
import showBaseDialog from './base-dialog'

export const dialog = (component, opts) => showBaseDialog(component, opts)

export const alert = (opts) => showAlertDialog(opts)
export const confirm = (opts) => showConfirmDialog(opts)
export const input = (opts) => showInputDialog(opts)
export const loading = (opts) => toggleLoadingDialog(opts)
export const open = (opts) => showOpenDialog(opts)

export default {
  dialog,
  open,
  alert,
  confirm,
  input,
  loading,
}
