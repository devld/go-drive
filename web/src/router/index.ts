import AppWrapper from '@/views/AppWrapper/index.vue'

import { T } from '@/i18n'
import { setTitle } from '@/utils'
import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'

import Home from '@/views/Home.vue'
import AdminIndex from '@/views/Admin/index.vue'

const routes: RouteRecordRaw[] = [
  {
    component: AppWrapper,
    path: '/',
    redirect: '/_/',
    children: [
      {
        name: 'Home',
        path: '/_/:path(.*)',
        component: Home,
        props: true,
      },
      {
        name: 'Admin',
        path: '/admin',
        component: AdminIndex,
        redirect: '/admin/site',
        children: [
          {
            name: 'SiteConfig',
            path: '/admin/site',
            component: () => import('@/views/Admin/Site.vue'),
            meta: { title: T('routes.title.site') },
          },
          {
            name: 'UsersManager',
            path: '/admin/users',
            component: () => import('@/views/Admin/Users.vue'),
            meta: { title: T('routes.title.users') },
          },
          {
            name: 'GroupsManager',
            path: '/admin/groups',
            component: () => import('@/views/Admin/Groups.vue'),
            meta: { title: T('routes.title.groups') },
          },
          {
            name: 'DrivesManager',
            path: '/admin/drives',
            component: () => import('@/views/Admin/Drives.vue'),
            meta: { title: T('routes.title.drives') },
          },
          {
            name: 'JobsManager',
            path: '/admin/jobs',
            component: () => import('@/views/Admin/Jobs.vue'),
            meta: { title: T('routes.title.jobs') },
          },
          {
            name: 'MiscSettings',
            path: '/admin/misc',
            component: () => import('@/views/Admin/Misc/index.vue'),
            meta: { title: T('routes.title.misc') },
          },
        ],
      },
    ],
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.afterEach((to) => {
  if (to.meta && to.meta.title) {
    setTitle(to.meta.title)
  }
})

export default router
