import { filenameExt } from '@/utils'
import MediaView from '@/views/HandlerViews/MediaView.vue'

export default {
  name: 'media',
  display: {
    name: 'Play',
    description: 'Play media',
    icon: '#icon-play-circle'
  },
  view: {
    name: 'MediaView',
    component: MediaView
  },
  supports: (entry) => entry.type === 'file' &&
    ['mp4', 'mp3', 'm4a', 'ogg'].includes(filenameExt(entry.name))
}
