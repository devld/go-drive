import { filenameExt } from '@/utils'

const TEXT_EDITOR_MAX_FILE_SIZE = 128 * 1024 // 128kb

export default {
  name: 'editor',
  display: (entry) => ({
    name: entry.meta.can_write ? 'Edit' : 'View',
    description: entry.meta.can_write ? 'Edit this file' : 'View this file'
  }),
  view: {
    name: 'TextEditView',
    component: () => import('./View.vue')
  },
  supports: (entry) => entry.type === 'file' && [
    'txt', 'md',
    'xml', 'html', 'css', 'scss', 'js', 'json', 'jsx', 'ts',
    'properties', 'yml', 'yaml', 'ini',
    'c', 'h', 'cpp',
    'go',
    'java', 'kt', 'gradle'
  ].includes(filenameExt(entry.name)) && entry.size <= TEXT_EDITOR_MAX_FILE_SIZE
}
