import { defineAsyncComponent } from 'vue'
import { T } from '@go-drive/i18n'
import { filenameExt } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'audio',
  display: {
    name: T('handler.audio.name'),
    description: T('handler.audio.desc'),
    icon: 'play-circle',
  },
  view: {
    name: 'AudioView',
    component: defineAsyncComponent(() => import('./AudioView.vue')),
  },
  supports: ({ entry }, { options }) =>
    entry.type === 'file' &&
    options['web.audioFileExts'].includes(filenameExt(entry.name)),
  order: 1000,
} as EntryHandler
