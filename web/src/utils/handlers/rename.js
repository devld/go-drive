import { moveEntry } from '@/api'
import { T } from '@/i18n'
import { dir, pathClean, pathJoin, taskDone, TASK_CANCELLED } from '..'

export default {
  name: 'rename',
  display: {
    name: T('handler.rename.name'),
    description: T('handler.rename.desc'),
    icon: '#icon-rename',
  },
  supports: (ctx, entry, parentEntry) =>
    entry.meta.writable && parentEntry && parentEntry.meta.writable,
  handler: (ctx, entry, { input, alert }) => {
    return new Promise((resolve) => {
      input({
        title: T('handler.rename.input_title'),
        text: entry.name,
        validator: {
          pattern: /^[^/\0:*"<>|]+$/,
          message: T('handler.rename.invalid_filename'),
        },
        onOk: async (text) => {
          if (text === entry.name) return
          try {
            await taskDone(
              moveEntry(
                ctx,
                entry.path,
                pathClean(pathJoin(dir(entry.path), text))
              )
            )
            resolve({ update: true })
          } catch (e) {
            if (e === TASK_CANCELLED) return
            alert(e.message)
            throw e
          }
        },
      })
    })
  },
}
