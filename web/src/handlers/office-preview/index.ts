import { T } from '@/i18n'
import { filenameExt } from '@/utils'
import { EntryHandler } from '../types'
import OfficePreviewView from './OfficePreviewView.vue'

export default {
  name: 'office',
  display: {
    name: T('handler.office.name'),
    description: T('handler.office.desc'),
    icon: '#icon-play-circle',
  },
  style: { fullscreen: true },
  view: {
    name: 'OfficePreviewView',
    component: OfficePreviewView,
  },
  supports: ({ entry }, { options }) =>
    options['web.officePreviewEnabled'] &&
    entry.type === 'file' &&
    ['docx', 'doc', 'xlsx', 'xls', 'pptx', 'ppt'].includes(
      filenameExt(entry.name)
    ),
} as EntryHandler
