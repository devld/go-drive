import { wrapAsyncComponent } from '@/components/async'
import { PREVIEW_FILE_MAX_SIZE } from '@/config'
import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'

export default {
  name: 'spreadsheet-preview',
  display: {
    name: T('preview.spreadsheet.name'),
    description: T('preview.spreadsheet.desc'),
    icon: 'document',
  },
  style: { fullscreen: true },
  view: {
    name: 'SpreadsheetPreviewView',
    component: wrapAsyncComponent(() => import('./PreviewView.vue')),
  },
  supports: ['file', 'csv,tsv', PREVIEW_FILE_MAX_SIZE],
  order: 10,
} as EntryHandler
