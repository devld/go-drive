/* eslint-disable quote-props */
import { filenameExt, mapOf } from '@/utils'
import DownloadView from '@/views/DownloadView'

const TEXT_EDITOR_MAX_FILE_SIZE = 128 * 1024 // 128kb

const HANDLERS = [
  {
    name: 'editor',
    view: {
      name: 'TextEditView',
      component: () => import('@/views/TextEditView')
    },
    supports: (entry, path, ext) => entry.type === 'file' && [
      'md', 'js', 'html', 'css', 'java', 'kt', 'json',
      'gradle', 'xml', 'properties', 'yml', 'yaml', 'ini'
    ].includes(ext) && entry.size <= TEXT_EDITOR_MAX_FILE_SIZE
  },
  {
    name: 'image',
    view: {
      name: 'ImageView',
      component: () => import('@/views/ImageView')
    },
    supports: (entry, path, ext) => entry.type === 'file' &&
      ['jpg', 'jpeg', 'png', 'gif'].includes(ext)
  },
  {
    name: 'download',
    view: {
      name: 'DownloadView',
      component: DownloadView
    },
    supports: (entry) => entry.type === 'file'
  }
]

export const HANDLER_COMPONENTS = mapOf(HANDLERS, h => h.view.name, h => h.view.component)

const HANDLERS_MAP = mapOf(HANDLERS, h => h.name)

export function getHandler (name) {
  return HANDLERS_MAP[name]
}

export function resolveEntryHandler (entry, path) {
  const ext = filenameExt(entry.name)
  const matches = []
  for (const h of HANDLERS) {
    if (h.supports(entry, path, ext)) matches.push(h)
  }
  return matches
}
