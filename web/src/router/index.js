import Vue from 'vue'
import VueRouter from 'vue-router'

import AppWrapper from '@/views/AppWrapper'

import Home from '@/views/Home'

Vue.use(VueRouter)

const routes = [
  {
    component: AppWrapper,
    path: '/',
    redirect: '/files/',
    children: [
      {
        name: 'Home',
        path: '/files/:path(.*)',
        component: Home,
        props: true
      }
    ]
  }
]

const router = new VueRouter({
  routes
})

export default router
