
import DownloadView from './View.vue'

export default {
  name: 'download',
  display: {
    name: 'Download',
    description: 'Download this file'
  },
  view: {
    name: 'DownloadView',
    component: DownloadView
  },
  supports: (entry) => entry.type === 'file'
}
