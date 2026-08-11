import { wrapAsyncComponent } from '@/components/async'
import { T } from '@/i18n'
import { filenameExt } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'office',
  display: {
    name: T('handler.office.name'),
    description: T('handler.office.desc'),
    icon: 'document',
  },
  style: { fullscreen: true },
  view: {
    name: 'OfficeView',
    component: wrapAsyncComponent(() => import('./OfficeView.vue')),
  },
  supports: ({ entry }, { config, user }) => {
    if (!user || entry.type !== 'file' || !config.wopi?.enabled) return false
    const actions = config.wopi.extensions[filenameExt(entry.name)]
    return !!actions && (actions.view || (entry.meta.writable && actions.edit))
  },
} as EntryHandler
