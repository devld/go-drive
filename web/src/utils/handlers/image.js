import { T } from '@/i18n'
import { filenameExt } from '@/utils'

export default {
  name: 'image',
  display: {
    name: T('handler.image.name'),
    description: T('handler.image.desc'),
    icon: '#icon-image',
  },
  view: {
    name: 'ImageView',
    component: () => import('@/views/HandlerViews/ImageView.vue'),
  },
  supports: entry =>
    entry.type === 'file' &&
    ['jpg', 'jpeg', 'png', 'gif'].includes(filenameExt(entry.name)),
}
