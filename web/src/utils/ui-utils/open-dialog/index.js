import OpenDialogInner from './OpenDialog.vue'
import showBaseDialog, { createDialog } from '../base-dialog'

const OpenDialog = createDialog('OpenDialog', OpenDialogInner)

export default function showInputDialog(Vue, opts) {
  return showBaseDialog(Vue, OpenDialog, { ...opts })
}
