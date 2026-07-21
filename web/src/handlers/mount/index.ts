import { mountPaths } from '@/api/admin'
import { T } from '@/i18n'
import { isAdmin } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'mount',
  display: {
    name: T('handler.mount.name'),
    description: T('handler.mount.desc'),
    icon: 'path',
  },
  supports: ({ entry }, { user }) =>
    isAdmin(user) && !entry.some((e) => e.meta.mountAt),
  multiple: true,
  handler({ entry: entries }, { open, loading, alert }) {
    return new Promise<{ update: true }>((resolve) =>
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
            resolve({ update: true })
          } catch (e: any) {
            alert(e.message)
            throw e
          } finally {
            loading()
          }
        },
      })
    )
  },
  order: 2002,
} as EntryHandler
