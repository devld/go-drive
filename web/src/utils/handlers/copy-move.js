import { copyEntry, deleteTask, moveEntry } from '@/api'
import { formatBytes, pathClean, pathJoin, taskDone, TASK_CANCELLED } from '..'

const createHandler = (isMove) => {
  return {
    name: isMove ? 'move' : 'copy',
    display: {
      name: (isMove ? 'Move' : 'Copy') + ' to',
      description: (isMove ? 'Move' : 'Copy') + ' files',
      icon: isMove ? '#icon-move' : '#icon-copy'
    },
    multiple: true,
    supports: isMove
      ? (entry, parentEntry) => (Array.isArray(entry)
        ? !entry.some(e => !e.meta.can_write)
        : entry.meta.can_write) &&
        parentEntry && parentEntry.meta.can_write
      : () => true,
    handler: (entries, { confirm, alert, loading, open }) => {
      if (!Array.isArray(entries)) entries = [entries]
      return new Promise((resolve) => {
        open({
          title: 'Select ' + (isMove ? 'move' : 'copy') + ' to', type: 'dir', filter: 'write',
          async onOk (path) {
            let override = true
            try {
              await confirm({
                message: 'Override or skip for duplicates?',
                confirmType: 'danger', confirmText: 'Override', cancelText: 'Skip'
              })
            } catch { override = false }
            let canceled = false
            let task
            const onCancel = () => {
              canceled = true
              return task && deleteTask(task.id)
            }
            try {
              for (const i in entries) {
                if (canceled) break
                const entry = entries[i]
                const dest = pathClean(pathJoin(path, entry.name))
                loading({ text: `${isMove ? 'Moving' : 'Copying'} ${entry.name}`, onCancel })
                const copyOrMove = isMove ? moveEntry : copyEntry
                await taskDone(
                  copyOrMove(entry.path, dest, override),
                  t => {
                    if (canceled) return false
                    task = t
                    loading({
                      text: `${isMove ? 'Moving' : 'Copying'} ${entry.name} ` +
                        `${formatBytes(task.progress.loaded)}/${formatBytes(task.progress.total)}`,
                      onCancel
                    })
                  }
                )
              }
              resolve({ update: true })
            } catch (e) {
              if (e === TASK_CANCELLED) return
              alert(e.message)
              throw e
            } finally {
              loading()
            }
          }
        })
      })
    }
  }
}

export const copy = createHandler(false)
export const move = createHandler(true)
