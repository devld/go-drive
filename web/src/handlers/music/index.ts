import { wrapAsyncComponent } from '@/components/async'
import { DEFAULT_AUDIO_FILE_EXTS } from '@/config'
import { T } from '@/i18n'
import { filenameExt } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'audio',
  display: {
    name: T('handler.audio.name'),
    description: T('handler.audio.desc'),
    icon: '#icon-play-circle',
  },
  view: {
    name: 'AudioView',
    component: wrapAsyncComponent(() => import('./AudioView.vue')),
  },
  supports: ({ entry }, { options }) =>
    entry.type === 'file' &&
    (options['web.audioFileExts'] || DEFAULT_AUDIO_FILE_EXTS).includes(
      filenameExt(entry.name)
    ),
  order: 1000,
} as EntryHandler
