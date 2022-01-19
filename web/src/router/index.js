import AppWrapper from '@/views/AppWrapper/index.vue'

import Home from '@/views/Home.vue'
import SharedFolder from '@/views/SharedFolder.vue'

import { setTitle } from '@/utils'
import { T } from '@/i18n'
import { createRouter, createWebHashHistory } from 'vue-router'
import { HOME_ROUTE_PREFIX, SHARED_FOLDER_ROUTE_PREFIX } from '@/config'

const routes = [
  {
    component: AppWrapper,
    path: '/',
    redirect: `${HOME_ROUTE_PREFIX}/`,
    children: [
      {
        name: 'Home',
        path: `${HOME_ROUTE_PREFIX}/:path(.*)`,
        component: Home,
        props: true,
      },
      {
        name: 'SharedFolder',
        path: `${SHARED_FOLDER_ROUTE_PREFIX}/:sharedId/:path(.*)`,
        component: SharedFolder,
        props: true,
      },
      {
        name: 'Admin',
        path: '/admin',
        component: () => import('@/views/Admin/index.vue'),
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
