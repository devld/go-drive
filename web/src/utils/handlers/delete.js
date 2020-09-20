import { deleteEntry, deleteTask } from '@/api'
import { taskDone } from '..'

export default {
  name: 'delete',
  display: {
    name: 'Delete',
    description: 'Delete this file',
    type: 'danger',
    icon: '#icon-delete'
  },
  supports: (entry) => Array.isArray(entry) ? !entry.some(e => !e.meta.can_write) : entry.meta.can_write,
  multiple: true,
  handler: async (entries, { confirm, alert, loading }) => {
    if (!Array.isArray(entries)) entries = [entries]
    try {
      await confirm({
        message: entries.length > 1 ? `Delete these ${entries.length} files?` : 'Delete this file?',
        confirmType: 'danger'
      })
    } catch { return }
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
        loading({ text: `Deleting ${entry.name}` })
        await taskDone(
          deleteEntry(entry.path),
          t => {
            task = t
            loading({
              text: `Deleting ${entry.name} ` +
                `${task.progress.loaded}/${task.progress.total}`,
              onCancel
            })
          }
        )
      }
      return { update: true }
    } catch (e) {
      alert(e.message)
    } finally {
      loading()
    }
  }
}
