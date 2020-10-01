import { mountPaths } from '@/api/admin'
import { isAdmin } from '..'

export default {
  name: 'mount',
  display: {
    name: 'Mount to',
    description: 'Mount entries to another location',
    icon: '#icon-path'
  },
  supports: (entry, user) => isAdmin(user) && !entry.meta.is_mount,
  multiple: true,
  async handler (entries, { open, loading }) {
    if (!Array.isArray(entries)) entries = [entries]
    open({
      title: 'Select mount to', type: 'dir',
      async onOk (path) {
        loading(true)
        try {
          await mountPaths(path, entries.map(e => ({ path: e.path, name: e.name })))
        } catch (e) {
          alert(e.message)
          throw e
        } finally {
          loading()
        }
      }
    })
  }
}
