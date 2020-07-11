import { filenameExt } from '@/utils'

export default {
  name: 'image',
  display: {
    name: 'Gallery',
    description: 'View images'
  },
  view: {
    name: 'ImageView',
    component: () => import('./View.vue')
  },
  supports: (entry) => entry.type === 'file' &&
    ['jpg', 'jpeg', 'png', 'gif'].includes(filenameExt(entry.name))
}
