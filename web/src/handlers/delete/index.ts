import { deleteEntry, deleteTask } from '@/api'
import { T } from '@/i18n'
import { Task } from '@/types'
import { taskDone, TASK_CANCELLED } from '@/utils'
import { EntryHandler } from '../types'

export default {
  name: 'delete',
  display: {
    name: T('handler.delete.name'),
    description: T('handler.delete.desc'),
    type: 'danger',
    icon: '#icon-delete',
  },
  supports: ({ entry, parent }) =>
    entry.every((e) => e.meta.writable) && parent?.meta.writable,
  multiple: true,
  handler: async ({ entry: entries }, { confirm, alert, loading }) => {
    try {
      await confirm({
        message:
          entries.length > 1
            ? T('handler.delete.confirm_n', { n: entries.length })
            : T('handler.delete.confirm'),
        confirmType: 'danger',
      })
    } catch {
      return
    }
    loading(true)
    let task: Task<void>
    let canceled = false
    const onCancel = () => {
      canceled = true
      return task && deleteTask(task.id)
    }
    try {
      for (const entry of entries) {
        if (canceled) break
        loading({ text: T('handler.delete.deleting', { n: entry.name }) })
        await taskDone(deleteEntry(entry.path), (t) => {
          if (canceled) return false
          task = t
          loading({
            text: T('handler.delete.deleting', {
              n: entry.name,
              p: task.progress
                ? `${task.progress.loaded}/${task.progress.total}`
                : '',
            }),
            onCancel,
          })
        })
      }
      return { update: true }
    } catch (e: any) {
      if (e === TASK_CANCELLED) return
      alert(e.message)
    } finally {
      loading()
    }
  },
  order: 2005,
} as EntryHandler
