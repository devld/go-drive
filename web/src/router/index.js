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
            component: () => import(/* webpackChunkName: "admin" */ '@/views/Admin/Users')
          },
          {
            name: 'GroupsManager',
            path: '/admin/groups',
            component: () => import(/* webpackChunkName: "admin" */ '@/views/Admin/Groups')
          },
          {
            name: 'DrivesManager',
            path: '/admin/drives',
            component: () => import(/* webpackChunkName: "admin" */ '@/views/Admin/Drives')
          }
        ]
      }
    ]
  }
]

const router = new VueRouter({
  routes
})

export default router
