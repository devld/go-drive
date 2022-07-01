import OpenDialogInner from './OpenDialog.vue'
import showBaseDialog, { BaseDialogOptions, createDialog } from '../base-dialog'
import { Entry } from '@/types'

export interface OpenDialogOptions extends BaseDialogOptions {
  type?: 'dir' | 'file'
  filter?: ((e: Entry) => boolean) | string
  max?: number
}

const OpenDialog = createDialog('OpenDialog', OpenDialogInner)

export default function showInputDialog(opts: OpenDialogOptions) {
  return showBaseDialog<string | Entry[]>(OpenDialog, { ...opts })
}
