import { createApp } from 'vue'
import App from './App.vue'

import router from './router'
import store from './store'
import i18n, { setLang } from './i18n'

import Components from '@/components'
import Utils from '@/utils'

import '@/styles/index.scss'
;(async () => {
  await setLang(navigator.language)

  createApp(App)
    .use(Utils)
    .use(Components)
    .use(router)
    .use(store)
    .use(i18n)
    .mount('#app')
})()
