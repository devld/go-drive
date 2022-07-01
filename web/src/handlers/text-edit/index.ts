import { wrapAsyncComponent } from '@/components/async'
import { T } from '@/i18n'
import { filenameExt } from '@/utils'
import { EntryHandler } from '../types'

const TEXT_EDITOR_MAX_FILE_SIZE = 128 * 1024 // 128kb

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
  view: {
    name: 'TextEditView',
    component: wrapAsyncComponent(() => import('./TextEditView.vue')),
  },
  supports: ({ entry }) =>
    entry.type === 'file' &&
    [
      'txt',
      'md',
      'xml',
      'html',
      'css',
      'scss',
      'js',
      'json',
      'jsx',
      'ts',
      'properties',
      'yml',
      'yaml',
      'ini',
      'c',
      'h',
      'cpp',
      'go',
      'java',
      'kt',
      'gradle',
      'ps1',
    ].includes(filenameExt(entry.name)) &&
    entry.size <= TEXT_EDITOR_MAX_FILE_SIZE,
} as EntryHandler
