
import { T } from '@/i18n'
import DownloadView from '@/views/HandlerViews/DownloadView.vue'

export default {
  name: 'download',
  display: {
    name: T('handler.download.name'),
    description: T('handler.download.desc'),
    icon: '#icon-download'
  },
  view: {
    name: 'DownloadView',
    component: DownloadView
  },
  multiple: true,
  supports: (entry) => Array.isArray(entry) ? !entry.some(e => e.type !== 'file') : entry.type === 'file'
}
