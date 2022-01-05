import { isT } from '@/i18n'
import { createApp } from 'vue'
import dialogUse from '../dialog-use'
import LoadingDialog from './LoadingDialog.vue'

let vm

export default function toggleLoadingDialog(opts) {
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
