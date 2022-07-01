import { T } from '@/i18n'
import { filenameExt } from '@/utils'
import { EntryHandler } from '../types'
import PDFPreviewView from './PDFPreviewView.vue'

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
  supports: ({ entry }) =>
    entry.type === 'file' && filenameExt(entry.name) === 'pdf',
} as EntryHandler
