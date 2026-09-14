import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'
import DownloadView from './DownloadView.vue'

export default {
  name: 'download',
  display: {
    name: T('handler.download.name'),
    description: T('handler.download.desc'),
    icon: 'download',
  },
  view: {
    name: 'DownloadView',
    component: DownloadView,
  },
  multiple: true,
  supports: ['file'],
  order: 2000,
} as EntryHandler
