import { defineAsyncComponent } from 'vue'
import { TEXT_EDITOR_MAX_FILE_SIZE } from '@/config'
import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'

export default {
  name: 'markdown',
  display: (entry) => ({
    name: T('handler.markdown.name'),
    description: T(
      entry.meta.writable
        ? 'handler.markdown.edit_desc'
        : 'handler.markdown.view_desc'
    ),
    icon: 'document',
  }),
  order: -1,
  style: { fullscreen: true },
  view: {
    name: 'MarkdownEditView',
    component: defineAsyncComponent(() => import('./MarkdownEditView.vue')),
  },
  supports: ['file', 'md,markdown', TEXT_EDITOR_MAX_FILE_SIZE],
} as EntryHandler
