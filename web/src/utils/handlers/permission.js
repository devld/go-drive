import { isAdmin } from '..'

export default {
  name: 'permission',
  display: {
    name: 'Permissions',
    description: 'Set permissions for this item',
    icon: '#icon-permission'
  },
  view: {
    name: 'PermissionsView',
    component: () => import(/* webpackChunkName: "admin" */ '@/views/HandlerViews/PermissionsView.vue')
  },
  supports: (entry, user) => isAdmin(user)
}
