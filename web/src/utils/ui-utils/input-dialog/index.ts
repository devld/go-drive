import InputDialogInner from './InputDialog.vue'
import showBaseDialog, { BaseDialogOptions, createDialog } from '../base-dialog'

const InputDialog = createDialog('InputDialog', InputDialogInner)

export type InputDialogValidateFunc = (v?: any) => PromiseValue<any>

export interface InputDialogValidator {
  trigger?: 'confirm' | 'change'
  validate?: InputDialogValidateFunc
  pattern?: RegExp
  message?: I18nText
}

export interface InputDialogOptions extends BaseDialogOptions {
  text?: string
  placeholder?: I18nText
  multipleLine?: boolean

  validator?: InputDialogValidator
}

export default function showInputDialog(opts: InputDialogOptions) {
  return showBaseDialog<string>(InputDialog, { ...opts })
}
