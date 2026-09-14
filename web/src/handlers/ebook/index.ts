import { wrapAsyncComponent } from '@/components/async'
import { PREVIEW_FILE_MAX_SIZE } from '@/config'
import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'

export default {
  name: 'ebook',
  display: {
    name: T('preview.ebook.name'),
    description: T('preview.ebook.desc'),
    icon: 'document',
  },
  style: { fullscreen: true },
  view: {
    name: 'EpubPreviewView',
    component: wrapAsyncComponent(() => import('./PreviewView.vue')),
  },
  supports: ['file', 'epub', PREVIEW_FILE_MAX_SIZE],
  order: 12,
} as EntryHandler
