import { entryMatches } from '@/utils'
import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'
import ArchiveView from './ArchiveView.vue'

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
  supports: ({ entry }, ctx) => {
    if (entry.type !== 'file') {
      return false
    }
    const config = ctx.config.artifact.archive
    if (!config || !entryMatches(entry, config.extensions)) {
      return false
    }
    const maxSize = config.maxSize
    return maxSize === undefined || entry.size < 0 || entry.size <= maxSize
  },
  order: -100,
} as EntryHandler
