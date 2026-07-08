import { wrapAsyncComponent } from '@/components/async'
import { T } from '@/i18n'
import { filenameExt } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'video',
  display: {
    name: T('handler.video.name'),
    description: T('handler.video.desc'),
    icon: 'play-circle',
  },
  style: { fullscreen: true },
  view: {
    name: 'VideoView',
    component: wrapAsyncComponent(() => import('./VideoView.vue')),
  },
  supports: ({ entry }, { options }) =>
    entry.type === 'file' &&
    options['web.videoFileExts'].includes(filenameExt(entry.name)),
  order: 1001,
} as EntryHandler
