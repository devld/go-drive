import Vue from 'vue'
import VueRouter from 'vue-router'

import Home from '@/views/Home'

import ErrorPage from '@/views/ErrorPage.vue'

Vue.use(VueRouter)

const routes = [
  {
    name: 'Index',
    path: '/',
    redirect: '/files/'
  },
  {
    name: 'Home',
    path: '/files/:path(.*)',
    component: Home,
    props: true
  },
  {
    name: 'ErrorPage',
    path: '/error/:code',
    component: ErrorPage,
    props: true
  }
]

const router = new VueRouter({
  routes
})

export default router
