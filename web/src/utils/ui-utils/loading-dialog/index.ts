import { SimpleButtonType } from '@/components/SimpleButton'
import { isT } from '@/i18n'
import { createApp } from 'vue'
import dialogUse from '../dialog-use'
import LoadingDialog from './LoadingDialog.vue'

export interface LoadingOptions {
  text?: I18nText
  onCancel?: () => PromiseValue<void>
  cancelText?: I18nText
  cancelType?: SimpleButtonType
}

let vm: any

export default function toggleLoadingDialog(opts?: LoadingOptions | boolean) {
  if (!vm) {
    const div = document.createElement('div')
    document.body.appendChild(div)

    vm = createApp(LoadingDialog).use(dialogUse).mount(div)
  }

  if (opts) {
    vm.show(typeof opts === 'object' && !isT(opts) ? opts : {})
  } else {
    vm.hide()
  }
}
