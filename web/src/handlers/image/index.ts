import { wrapAsyncComponent } from '@/components/async'
import { T } from '@/i18n'
import { filenameExt } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'image',
  display: {
    name: T('handler.image.name'),
    description: T('handler.image.desc'),
    icon: '#icon-image',
  },
  view: {
    name: 'ImageView',
    component: wrapAsyncComponent(() => import('./ImageView.vue')),
  },
  supports: ({ entry }) =>
    entry.type === 'file' &&
    ['jpg', 'jpeg', 'png', 'gif'].includes(filenameExt(entry.name)),
} as EntryHandler
