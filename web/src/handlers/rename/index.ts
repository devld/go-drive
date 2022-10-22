import { moveEntry } from '@/api'
import { T } from '@/i18n'
import { dir, pathClean, pathJoin, taskDone, TASK_CANCELLED } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'rename',
  display: {
    name: T('handler.rename.name'),
    description: T('handler.rename.desc'),
    icon: '#icon-rename',
  },
  supports: ({ entry, parent }) => entry.meta.writable && parent?.meta.writable,
  handler: ({ entry }, { input, alert }) => {
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
              moveEntry(entry.path, pathClean(pathJoin(dir(entry.path), text)))
            )
            resolve({ update: true })
          } catch (e: any) {
            if (e === TASK_CANCELLED) return
            alert(e.message)
            throw e
          }
        },
      })
    })
  },
  order: 2005,
} as EntryHandler
