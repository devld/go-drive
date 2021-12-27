import { T } from '@/i18n'
import PdfView from '@/views/HandlerViews/PdfView.vue'
import { filenameExt } from '@/utils'

export default {
  name: 'pdf',
  display: {
    name: T('handler.pdf.name'),
    description: T('handler.pdf.desc'),
    icon: '#icon-wendang',
  },
  view: {
    name: 'PdfView',
    component: PdfView,
  },
  supports: entry => entry.type === 'file' && filenameExt(entry.name) === 'pdf',
}
