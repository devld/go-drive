import { T } from '@/i18n'
import { filenameExt } from '@/utils'
import OfficePreviewView from '@/views/HandlerViews/OfficePreviewView.vue'

export default {
  name: 'office',
  display: {
    name: T('handler.office.name'),
    description: T('handler.office.desc'),
    icon: '#icon-play-circle',
  },
  view: {
    name: 'OfficePreviewView',
    component: OfficePreviewView,
  },
  supports: (entry, parentEntry, { options }) =>
    options['web.officePreviewEnabled'] &&
    entry.type === 'file' &&
    ['docx', 'doc', 'xlsx', 'xls', 'pptx', 'ppt'].includes(
      filenameExt(entry.name)
    ),
}
