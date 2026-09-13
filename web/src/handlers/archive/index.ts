import { filenameExt } from '@/utils'
import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'
import ArchiveView from './ArchiveView.vue'

const ARCHIVE_EXTENSIONS = new Set(['zip', '7z', 'rar'])

export default {
  name: 'archive',
  display: {
    name: T('handler.archive.name'),
    description: T('handler.archive.desc'),
    icon: 'archive',
  },
  view: {
    name: 'ArchiveView',
    component: ArchiveView,
  },
  supports: ({ entry }) =>
    entry.type === 'file' && ARCHIVE_EXTENSIONS.has(filenameExt(entry.name)),
  order: -100,
} as EntryHandler
