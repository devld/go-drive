import { deleteEntry, deleteTask } from '@/api'
import { T } from '@/i18n'
import { taskDone, TASK_CANCELLED } from '..'

export default {
  name: 'delete',
  display: {
    name: T('handler.delete.name'),
    description: T('handler.delete.desc'),
    type: 'danger',
    icon: '#icon-delete',
  },
  supports: (entry, parentEntry) =>
    (Array.isArray(entry)
      ? !entry.some(e => !e.meta.can_write)
      : entry.meta.can_write) &&
    parentEntry &&
    parentEntry.meta.can_write,
  multiple: true,
  handler: async (entries, { confirm, alert, loading }) => {
    if (!Array.isArray(entries)) entries = [entries]
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
    let task
    let canceled = false
    const onCancel = () => {
      canceled = true
      return task && deleteTask(task.id)
    }
    try {
      for (const entry of entries) {
        if (canceled) break
        loading({ text: T('handler.delete.deleting', { n: entry.name }) })
        await taskDone(deleteEntry(entry.path), t => {
          if (canceled) return false
          task = t
          loading({
            text: T('handler.delete.deleting', {
              n: entry.name,
              p: `${task.progress.loaded}/${task.progress.total}`,
            }),
            onCancel,
          })
        })
      }
      return { update: true }
    } catch (e) {
      if (e === TASK_CANCELLED) return
      alert(e.message)
    } finally {
      loading()
    }
  },
}
