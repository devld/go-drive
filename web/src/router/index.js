import Vue from 'vue'
import VueRouter from 'vue-router'

import AppWrapper from '@/views/AppWrapper'

import Home from '@/views/Home'
import { setTitle } from '@/utils'

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
      },
      {
        name: 'Admin',
        path: '/admin',
        component: () => import(/* webpackChunkName: "admin" */ '@/views/Admin'),
        redirect: '/admin/users',
        children: [
          {
            name: 'UsersManager',
            path: '/admin/users',
            component: () => import(/* webpackChunkName: "admin" */ '@/views/Admin/Users'),
            meta: { title: 'Users' }
          },
          {
            name: 'GroupsManager',
            path: '/admin/groups',
            component: () => import(/* webpackChunkName: "admin" */ '@/views/Admin/Groups'),
            meta: { title: 'Groups' }
          },
          {
            name: 'DrivesManager',
            path: '/admin/drives',
            component: () => import(/* webpackChunkName: "admin" */ '@/views/Admin/Drives'),
            meta: { title: 'Drives' }
          },
          {
            name: 'MiscSettings',
            path: '/admin/misc',
            component: () => import(/* webpackChunkName: "admin" */ '@/views/Admin/Misc'),
            meta: { title: 'Misc' }
          }
        ]
      }
    ]
  }
]

const router = new VueRouter({
  routes
})

// see https://github.com/vuejs/vue-router/issues/1849#issuecomment-340767577
// detect it's IE11
if ('-ms-scroll-limit' in document.documentElement.style && '-ms-ime-align' in document.documentElement.style) {
  window.addEventListener('hashchange', function (event) {
    const currentPath = window.location.hash.slice(1)
    if (router.currentRoute !== currentPath) {
      router.push(currentPath)
    }
  }, false)
}

router.afterEach((to) => {
  if (to.meta && to.meta.title) {
    setTitle(to.meta.title)
  }
})

export default router
