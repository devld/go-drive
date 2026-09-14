import { wrapAsyncComponent } from '@/components/async'
import { PREVIEW_FILE_MAX_SIZE } from '@/config'
import { T } from '@go-drive/i18n'
import { EntryHandler } from '../types'

export default {
  name: 'model3d-preview',
  display: {
    name: T('preview.model3d.name'),
    description: T('preview.model3d.desc'),
    icon: 'document',
  },
  style: { fullscreen: true },
  view: {
    name: 'Model3dPreviewView',
    component: wrapAsyncComponent(() => import('./PreviewView.vue')),
  },
  supports: ['file', 'glb,gltf', PREVIEW_FILE_MAX_SIZE],
  order: 14,
} as EntryHandler
