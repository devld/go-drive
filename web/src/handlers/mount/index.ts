import { mountPaths } from '@/api/admin'
import { T } from '@/i18n'
import { isAdmin } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'mount',
  display: {
    name: T('handler.mount.name'),
    description: T('handler.mount.desc'),
    icon: '#icon-path',
  },
  supports: ({ entry }, { user }) =>
    isAdmin(user) && !entry.some((e) => e.meta.mountAt),
  multiple: true,
  async handler({ entry: entries }, { open, loading, alert }) {
    open({
      title: T('handler.mount.open_title'),
      type: 'dir',
      async onOk(destEntry) {
        loading(true)
        try {
          await mountPaths(
            destEntry.path,
            entries.map((e) => ({ path: e.path, name: e.name }))
          )
        } catch (e: any) {
          alert(e.message)
          throw e
        } finally {
          loading()
        }
      },
    })
  },
  order: 2002,
} as EntryHandler
