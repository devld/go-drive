import { moveEntry } from '@/api'
import { dir, pathClean, pathJoin, taskDone } from '..'

export default {
  name: 'rename',
  display: {
    name: 'Rename',
    description: 'Rename this file',
    icon: '#icon-rename'
  },
  supports: (entry) => entry.meta.can_write,
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
            alert(e.message)
            throw e
          }
        }
      })
    })
  }
}
