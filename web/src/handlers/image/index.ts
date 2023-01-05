import { wrapAsyncComponent } from '@/components/async'
import { DEFAULT_IMAGE_FILE_EXTS } from '@/config'
import { T } from '@/i18n'
import { filenameExt } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'image',
  display: {
    name: T('handler.image.name'),
    description: T('handler.image.desc'),
    icon: '#icon-image',
  },
  style: { fullscreen: true },
  view: {
    name: 'ImageView',
    component: wrapAsyncComponent(() => import('./ImageView.vue')),
  },
  supports: ({ entry }, { options }) =>
    entry.type === 'file' &&
    (options['web.imageFileExts'] || DEFAULT_IMAGE_FILE_EXTS).includes(
      filenameExt(entry.name)
    ),
} as EntryHandler
