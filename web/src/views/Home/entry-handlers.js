/* eslint-disable quote-props */
import { filenameExt, mapOf } from '@/utils'

const HANDLERS = [
  {
    name: 'editor',
    view: {
      name: 'TextEditView',
      component: () => import('@/views/TextEditView')
    },
    supports: (entry, path, ext) => entry.type === 'file' && [
      'md', 'js', 'html', 'css', 'java', 'kt', 'json',
      'gradle', 'xml', 'properties', 'yml', 'yaml'
    ].includes(ext)
  },
  {
    name: 'image',
    view: {
      name: 'ImageView',
      component: () => import('@/views/ImageView')
    },
    supports: (entry, path, ext) => entry.type === 'file' &&
      ['jpg', 'jpeg', 'png', 'gif'].includes(ext)
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
