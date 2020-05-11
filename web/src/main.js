import Vue from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'

import Components from '@/components'
import Utils from '@/utils'

import '@/styles/index.scss'

Vue.config.productionTip = false

Vue.use(Components)
Vue.use(Utils)

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount('#app')
