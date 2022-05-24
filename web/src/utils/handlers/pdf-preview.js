import { T } from '@/i18n'
import { filenameExt } from '@/utils'
import PDFPreviewView from '@/views/HandlerViews/PDFPreviewView.vue'

export default {
  name: 'pdf',
  display: {
    name: T('handler.pdf.name'),
    description: T('handler.pdf.desc'),
    icon: '#icon-play-circle',
  },
  view: {
    name: 'PDFPreviewView',
    component: PDFPreviewView,
  },
  supports: (entry) =>
    entry.type === 'file' && filenameExt(entry.name) === 'pdf',
}
