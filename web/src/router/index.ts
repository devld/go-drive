import AppWrapper from '@/views/AppWrapper/index.vue'

import { T } from '@/i18n'
import { setTitle } from '@/utils'
import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'
import { EXPLORER_PATH_BASE } from '@/config'

import Home from '@/views/Home.vue'
import AdminIndex from '@/views/Admin/index.vue'

const routes: RouteRecordRaw[] = [
  {
    component: AppWrapper,
    path: '/',
    redirect: `${EXPLORER_PATH_BASE}/`,
    children: [
      {
        name: 'Home',
        path: `${EXPLORER_PATH_BASE}/:path(.*)`,
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
            name: 'ExtraDrivesManager',
            path: '/admin/extra-drives',
            component: () => import('@/views/Admin/ExtraDrives/index.vue'),
            meta: { title: T('routes.title.extra_drives') },
          },
          {
            name: 'JobsManager',
            path: '/admin/jobs',
            component: () => import('@/views/Admin/Jobs/index.vue'),
            meta: { title: T('routes.title.jobs') },
          },
          {
            name: 'PathMetaManager',
            path: '/admin/path-meta',
            component: () => import('@/views/Admin/PathMeta.vue'),
            meta: { title: T('routes.title.path_meta') },
          },
          {
            name: 'FileBucketsManager',
            path: '/admin/file-buckets',
            component: () => import('@/views/Admin/FileBuckets.vue'),
            meta: { title: T('routes.title.file_buckets') },
          },
          {
            name: 'MiscSettings',
            path: '/admin/misc',
            component: () => import('@/views/Admin/Misc/index.vue'),
            meta: { title: T('routes.title.misc') },
          },
          {
            name: 'SysStats',
            path: '/admin/stats',
            component: () => import('@/views/Admin/SysStats.vue'),
            meta: { title: T('routes.title.statistics') },
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
