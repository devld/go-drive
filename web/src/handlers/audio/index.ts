import { T } from '@go-drive/i18n'
import { filenameExt } from '@/utils'
import { EntryHandler } from '../types'
import AudioView from './AudioView.vue'

export default {
  name: 'audio',
  display: {
    name: T('handler.audio.name'),
    description: T('handler.audio.desc'),
    icon: 'play-circle',
  },
  view: {
    name: 'AudioView',
    component: AudioView,
  },
  supports: ({ entry }, { options }) =>
    entry.type === 'file' &&
    options['web.audioFileExts'].includes(filenameExt(entry.name)),
  order: 1000,
} as EntryHandler
