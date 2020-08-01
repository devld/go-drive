
import DownloadView from '@/views/HandlerViews/DownloadView.vue'

export default {
  name: 'download',
  display: {
    name: 'Download',
    description: 'Download this file',
    icon: '#icon-download'
  },
  view: {
    name: 'DownloadView',
    component: DownloadView
  },
  supports: (entry) => entry.type === 'file'
}
