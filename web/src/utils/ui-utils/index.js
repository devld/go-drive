
import { showAlertDialog, showConfirmDialog } from './text-dialog'
import showInputDialog from './input-dialog'
import toggleLoadingDialog from './loading-dialog'

export default {
  install (Vue) {
    const alert = (opts) => showAlertDialog(Vue, opts)
    const confirm = (opts) => showConfirmDialog(Vue, opts)
    const input = (opts) => showInputDialog(Vue, opts)
    const loading = (opts) => toggleLoadingDialog(Vue, opts)

    Vue.prototype.$alert = alert
    Vue.prototype.$confirm = confirm
    Vue.prototype.$input = input
    Vue.prototype.$loading = loading

    Vue.prototype.$uiUtils = {
      alert, confirm, input, loading
    }
  }
}
