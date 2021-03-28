import i18n, { isT } from '@/i18n'
import LoadingDialog from './LoadingDialog.vue'

let vm

export default function toggleLoadingDialog(Vue, opts) {
  if (!vm) {
    const div = document.createElement('div')
    document.body.appendChild(div)
    vm = new Vue({ i18n, ...LoadingDialog })
    vm.$mount(div)
  }

  if (opts) {
    vm.show(typeof opts === 'object' && !isT(opts) ? opts : {})
  } else {
    vm.hide()
  }
}
