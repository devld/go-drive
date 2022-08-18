import { DEFAULT_VIDEO_FILE_EXTS } from '@/config'
import { T } from '@/i18n'
import { filenameExt } from '@/utils'
import { EntryHandler } from '../types'
import VideoView from './VideoView.vue'

export default {
  name: 'video',
  display: {
    name: T('handler.video.name'),
    description: T('handler.video.desc'),
    icon: '#icon-play-circle',
  },
  view: {
    name: 'VideoView',
    component: VideoView,
  },
  supports: ({ entry }, { options }) =>
    entry.type === 'file' &&
    (options['web.videoFileExts'] || DEFAULT_VIDEO_FILE_EXTS).includes(
      filenameExt(entry.name)
    ),
  order: 1001,
} as EntryHandler
