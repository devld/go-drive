import AppWrapper from '@/views/AppWrapper/index.vue'

import Home from '@/views/Home.vue'
import { setTitle } from '@/utils'
import { T } from '@/i18n'
import { createRouter, createWebHashHistory } from 'vue-router'
import { wrapAsyncComponent } from '@/components/async'

const routes = [
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
        component: wrapAsyncComponent(() => import('@/views/Admin/index.vue')),
        redirect: '/admin/site',
        children: [
          {
            name: 'SiteConfig',
            path: '/admin/site',
            component: wrapAsyncComponent(() =>
              import('@/views/Admin/Site.vue')
            ),
            meta: { title: T('routes.title.site') },
          },
          {
            name: 'UsersManager',
            path: '/admin/users',
            component: wrapAsyncComponent(() =>
              import('@/views/Admin/Users.vue')
            ),
            meta: { title: T('routes.title.users') },
          },
          {
            name: 'GroupsManager',
            path: '/admin/groups',
            component: wrapAsyncComponent(() =>
              import('@/views/Admin/Groups.vue')
            ),
            meta: { title: T('routes.title.groups') },
          },
          {
            name: 'DrivesManager',
            path: '/admin/drives',
            component: wrapAsyncComponent(() =>
              import('@/views/Admin/Drives.vue')
            ),
            meta: { title: T('routes.title.drives') },
          },
          {
            name: 'MiscSettings',
            path: '/admin/misc',
            component: wrapAsyncComponent(() =>
              import('@/views/Admin/Misc/index.vue')
            ),
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
