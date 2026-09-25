import { wrapAsyncComponent } from '@/components/async'
import { PREVIEW_FILE_MAX_SIZE } from '@/config'
import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'

export default {
  name: 'markdown',
  display: {
    name: T('preview.markdown.name'),
    description: T('preview.markdown.desc'),
    icon: 'document',
  },
  style: { fullscreen: true },
  view: {
    name: 'MarkdownView',
    component: wrapAsyncComponent(() => import('./PreviewView.vue')),
  },
  supports: ['file', 'md,markdown', PREVIEW_FILE_MAX_SIZE],
  order: -1,
} as EntryHandler
