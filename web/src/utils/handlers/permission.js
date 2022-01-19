import { T } from '@/i18n'
import { defineAsyncComponent } from 'vue'
import { isAdmin, isShared } from '..'

export default {
  name: 'permission',
  display: {
    name: T('handler.permission.name'),
    description: T('handler.permission.desc'),
    icon: '#icon-permission',
  },
  view: {
    name: 'PermissionsView',
    component: defineAsyncComponent(() =>
      import('@/views/HandlerViews/PermissionsView.vue')
    ),
  },
  supports: (ctx, entry, parentEntry, user) => !isShared(ctx) && isAdmin(user),
}
