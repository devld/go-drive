import { moveEntry } from '@/api'
import { dir, pathClean, pathJoin, taskDone, TASK_CANCELLED } from '..'

export default {
  name: 'rename',
  display: {
    name: 'Rename',
    description: 'Rename this file',
    icon: '#icon-rename'
  },
  supports: (entry, parentEntry) => entry.meta.can_write &&
    parentEntry && parentEntry.meta.can_write,
  handler: (entry, { input, alert }) => {
    return new Promise((resolve) => {
      input({
        title: 'Rename',
        text: entry.name,
        validator: {
          pattern: /^[^/\0:*"<>|]+$/,
          message: 'Invalid filename'
        },
        onOk: async text => {
          if (text === entry.name) return
          try {
            await taskDone(moveEntry(entry.path, pathClean(pathJoin(dir(entry.path), text))))
            resolve({ update: true })
          } catch (e) {
            if (e === TASK_CANCELLED) return
            alert(e.message)
            throw e
          }
        }
      })
    })
  }
}
