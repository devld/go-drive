import { PREVIEW_FILE_MAX_SIZE } from '@/config'
import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'
import PreviewView from './PreviewView.vue'

export default {
  name: 'web-preview',
  display: {
    name: T('preview.web.name'),
    description: T('preview.web.desc'),
    icon: 'document',
  },
  style: { fullscreen: true },
  view: {
    name: 'WebPreviewView',
    component: PreviewView,
  },
  supports: ['file', 'html,htm,svg', PREVIEW_FILE_MAX_SIZE],
  order: 11,
} as EntryHandler
