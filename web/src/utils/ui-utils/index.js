import { showAlertDialog, showConfirmDialog } from './text-dialog'
import showInputDialog from './input-dialog'
import toggleLoadingDialog from './loading-dialog'
import showOpenDialog from './open-dialog'
import showBaseDialog from './base-dialog'

export default {
  install(Vue) {
    const dialog = (component, opts) => showBaseDialog(Vue, component, opts)

    const alert = opts => showAlertDialog(Vue, opts)
    const confirm = opts => showConfirmDialog(Vue, opts)
    const input = opts => showInputDialog(Vue, opts)
    const loading = opts => toggleLoadingDialog(Vue, opts)
    const open = opts => showOpenDialog(Vue, opts)

    Vue.prototype.$dialog = dialog
    Vue.prototype.$alert = alert
    Vue.prototype.$confirm = confirm
    Vue.prototype.$input = input
    Vue.prototype.$loading = loading
    Vue.prototype.$open = open

    Vue.prototype.$uiUtils = {
      dialog,
      open,
      alert,
      confirm,
      input,
      loading,
    }
  },
}
