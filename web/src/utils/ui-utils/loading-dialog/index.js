import LoadingDialog from './LoadingDialog.vue'

let vm

export default function toggleLoadingDialog (Vue, opts) {
  if (!vm) {
    const div = document.createElement('div')
    document.body.appendChild(div)
    vm = new Vue(LoadingDialog)
    vm.$mount(div)
  }

  if (opts) {
    vm.show(typeof (opts) === 'object' ? opts : {})
  } else {
    vm.hide()
  }
}
