import { deleteEntry } from '@/api'

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
  handler: async (entry, { confirm, alert, loading }) => {
    if (!Array.isArray(entry)) entry = [entry]
    try {
      await confirm({
        message: entry.length > 1 ? `Delete these ${entry.length} files?` : 'Delete this file?',
        confirmType: 'danger'
      })
    } catch { return }
    loading(true)
    try {
      let canceled = false
      for (const i in entry) {
        if (canceled) break
        const e = entry[i]
        loading({
          text: `deleting ${i + 1}/${entry.length}`,
          onCancel: () => { canceled = true }
        })
        await deleteEntry(e.path)
      }
      return { update: true }
    } catch (e) {
      alert(e.message)
    } finally {
      loading()
    }
  }
}
