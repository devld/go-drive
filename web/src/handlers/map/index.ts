import { wrapAsyncComponent } from '@/components/async'
import { PREVIEW_FILE_MAX_SIZE } from '@/config'
import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'

export default {
  name: 'map',
  display: {
    name: T('preview.map.name'),
    description: T('preview.map.desc'),
    icon: 'document',
  },
  style: { fullscreen: true },
  view: {
    name: 'MapPreviewView',
    component: wrapAsyncComponent(() => import('./PreviewView.vue')),
  },
  supports: ['file', 'geojson,gpx,kml', PREVIEW_FILE_MAX_SIZE],
  order: 15,
} as EntryHandler
