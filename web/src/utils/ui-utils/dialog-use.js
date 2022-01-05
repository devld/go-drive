import Utils from '@/utils'
import Components from '@/components'
import i18n from '@/i18n'
import router from '@/router'
import store from '@/store'

export default {
  /**
   * @param {import("vue").App} app
   */
  install(app) {
    app.use(Utils).use(Components).use(i18n).use(router).use(store)
  },
}
