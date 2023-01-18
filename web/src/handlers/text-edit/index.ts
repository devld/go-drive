import { wrapAsyncComponent } from '@/components/async'
import { DEFAULT_TEXT_FILE_EXTS, TEXT_EDITOR_MAX_FILE_SIZE } from '@/config'
import { T } from '@/i18n'
import { entryMatches, filenameExt } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'editor',
  display: (entry) => ({
    name: T(
      entry.meta.writable
        ? 'handler.text_edit.edit_name'
        : 'handler.text_edit.view_name'
    ),
    description: T(
      entry.meta.writable
        ? 'handler.text_edit.edit_desc'
        : 'handler.text_edit.view_desc'
    ),
    icon: '#icon-cursor-text',
  }),
  style: { fullscreen: true },
  view: {
    name: 'TextEditView',
    component: wrapAsyncComponent(() => import('./TextEditView.vue')),
  },
  supports: ({ entry }, { options }) =>
    entry.type === 'file' &&
    entryMatches(
      entry,
      options['web.textFileExts'] || DEFAULT_TEXT_FILE_EXTS
    ) &&
    entry.size <= TEXT_EDITOR_MAX_FILE_SIZE,
} as EntryHandler
