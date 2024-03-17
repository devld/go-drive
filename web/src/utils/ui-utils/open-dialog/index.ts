import OpenDialogInner from './OpenDialog.vue'
import showBaseDialog, { BaseDialogOptions, createDialog } from '../base-dialog'
import { Entry } from '@/types'

export interface OpenDialogOptions<T extends Entry | Entry[] = Entry | Entry[]>
  extends BaseDialogOptions<T> {
  type?: 'dir' | 'file'
  filter?: ((e: Entry) => boolean) | string
  max?: number
}

const OpenDialog = createDialog('OpenDialog', OpenDialogInner)

export interface OpenDialogDirOptions extends OpenDialogOptions<Entry> {
  type: 'dir'
}

export interface OpenDialogFileOptions extends OpenDialogOptions<Entry[]> {
  type: 'file'
}

function showOpenDialog(opts: OpenDialogDirOptions): Promise<Entry>
function showOpenDialog(opts: OpenDialogFileOptions): Promise<Entry[]>
function showOpenDialog(opts: OpenDialogDirOptions | OpenDialogFileOptions) {
  return showBaseDialog<Entry | Entry[]>(OpenDialog, { ...(opts as any) })
}

export default showOpenDialog