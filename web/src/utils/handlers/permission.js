import { T } from '@/i18n'
import { isAdmin } from '..'

export default {
  name: 'permission',
  display: {
    name: T('handler.permission.name'),
    description: T('handler.permission.desc'),
    icon: '#icon-permission'
  },
  view: {
    name: 'PermissionsView',
    component: () => import(/* webpackChunkName: "admin" */ '@/views/HandlerViews/PermissionsView.vue')
  },
  supports: (entry, parentEntry, user) => isAdmin(user)
}
