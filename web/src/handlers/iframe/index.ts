import { T } from '@/i18n'
import { entryMatches } from '@/utils'
import { EntryHandler } from '../types'
import IframePreviewView from './IframePreviewView.vue'

export default {
  name: 'iframe',
  display: {
    name: T('handler.iframe.name'),
    description: T('handler.iframe.desc'),
    icon: '#icon-wendang',
  },
  style: { fullscreen: true },
  view: {
    name: 'IframePreviewView',
    component: IframePreviewView,
  },
  supports: ({ entry }, { options }) =>
    entry.type === 'file' &&
    options['web.externalFileViewers'].some((e) => entryMatches(entry, e.exts)),
} as EntryHandler
