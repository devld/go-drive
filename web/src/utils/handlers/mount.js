import { mountPaths } from '@/api/admin'
import { T } from '@/i18n'
import { isAdmin } from '..'

export default {
  name: 'mount',
  display: {
    name: T('handler.mount.name'),
    description: T('handler.mount.desc'),
    icon: '#icon-path',
  },
  supports: (entry, parentEntry, user) =>
    isAdmin(user) &&
    (Array.isArray(entry)
      ? !entry.some(e => e.meta.is_mount)
      : !entry.meta.is_mount),
  multiple: true,
  async handler(entries, { open, loading }) {
    if (!Array.isArray(entries)) entries = [entries]
    open({
      title: T('handler.mount.open_title'),
      type: 'dir',
      async onOk(path) {
        loading(true)
        try {
          await mountPaths(
            path,
            entries.map(e => ({ path: e.path, name: e.name }))
          )
        } catch (e) {
          alert(e.message)
          throw e
        } finally {
          loading()
        }
      },
    })
  },
}
