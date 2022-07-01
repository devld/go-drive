import { wrapAsyncComponent } from '@/components/async'
import { T } from '@/i18n'
import { isAdmin } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'permission',
  display: {
    name: T('handler.permission.name'),
    description: T('handler.permission.desc'),
    icon: '#icon-permission',
  },
  view: {
    name: 'PermissionsView',
    component: wrapAsyncComponent(() => import('./PermissionsView.vue')),
  },
  supports: (_, { user }) => isAdmin(user),
  order: 2001,
} as EntryHandler
