import { filenameExt } from '@/utils'

export default {
  name: 'image',
  display: {
    name: 'Gallery',
    description: 'View images',
    icon: '#icon-image'
  },
  view: {
    name: 'ImageView',
    component: () => import('@/views/HandlerViews/ImageView.vue')
  },
  supports: (entry) => entry.type === 'file' &&
    ['jpg', 'jpeg', 'png', 'gif'].includes(filenameExt(entry.name))
}
