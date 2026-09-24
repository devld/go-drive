import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'
import DownloadView, { downloadFiles } from './DownloadView.vue'

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
  supports: ({ entry }) => entry.every((e) => e.type === 'file'),
  order: 2000,
  handler: async ({ entry }, _, { source }) => {
    if (source === 'entry') return { view: true }

    downloadFiles(entry)
  },
} as EntryHandler
