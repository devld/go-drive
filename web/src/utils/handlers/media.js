import { T } from '@/i18n'
import { filenameExt } from '@/utils'
import MediaView from '@/views/HandlerViews/MediaView.vue'

export default {
  name: 'media',
  display: {
    name: T('handler.media.name'),
    description: T('handler.media.desc'),
    icon: '#icon-play-circle'
  },
  view: {
    name: 'MediaView',
    component: MediaView
  },
  supports: entry =>
    entry.type === 'file' &&
    ['mp4', 'mp3', 'm4a', 'ogg'].includes(filenameExt(entry.name))
}
