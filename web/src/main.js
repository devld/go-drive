import Vue from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import i18n, { setLang } from './i18n'

import Components from '@/components'
import Utils from '@/utils'

import '@/styles/index.scss'
import './registerServiceWorker'

Vue.config.productionTip = false

Vue.use(Components)
Vue.use(Utils)

;(async () => {
  await setLang(navigator.language)

  new Vue({
    router,
    store,
    i18n,
    render: h => h(App)
  }).$mount('#app')
})()
