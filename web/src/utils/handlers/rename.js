import { moveEntry } from '@/api'
import { dir, pathClean, pathJoin, taskDone, wait } from '..'

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
          pattern: /^[^/\\0:*"<>|]+$/,
          message: 'Invalid filename'
        },
        onOk: async text => {
          if (text === entry.name) return
          try {
            let task = await moveEntry(entry.path, pathClean(pathJoin(dir(entry.path), text)))
            task = await taskDone(task, () => wait(1000))
            if (task.status === 'done') {
              resolve({ update: true })
            } else if (task.status === 'error') {
              alert(task.error.message)
              throw task.error
            }
          } catch (e) {
            alert(e.message)
            throw e
          }
        }
      })
    })
  }
}
