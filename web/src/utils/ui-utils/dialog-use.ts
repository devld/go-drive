import Utils from '@/utils'
import Components from '@/components'
import i18n from '@/i18n'
import router from '@/router'
import store from '@/store'
import { Plugin } from 'vue'

export default {
  install(app) {
    app.use(Utils).use(Components).use(i18n).use(router).use(store)
  },
} as Plugin
