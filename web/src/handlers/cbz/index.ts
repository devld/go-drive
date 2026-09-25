import { wrapAsyncComponent } from '@/components/async'
import { PREVIEW_FILE_MAX_SIZE } from '@/config'
import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'

export default {
  name: 'cbz',
  display: {
    name: T('preview.cbz.name'),
    description: T('preview.cbz.desc'),
    icon: 'archive',
  },
  style: { fullscreen: true },
  view: {
    name: 'CbzPreviewView',
    component: wrapAsyncComponent(() => import('./PreviewView.vue')),
  },
  supports: ['file', 'cbz', PREVIEW_FILE_MAX_SIZE],
  order: 13,
} as EntryHandler
