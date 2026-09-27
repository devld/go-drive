import { defineAsyncComponent } from 'vue'
import { T } from '@go-drive/i18n'
import { filenameExt } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'image',
  display: {
    name: T('handler.image.name'),
    description: T('handler.image.desc'),
    icon: 'image',
  },
  style: { fullscreen: true },
  view: {
    name: 'ImageView',
    component: defineAsyncComponent(() => import('./ImageView.vue')),
  },
  supports: ({ entry }, { options }) =>
    entry.type === 'file' &&
    options['web.imageFileExts'].includes(filenameExt(entry.name)),
} as EntryHandler
