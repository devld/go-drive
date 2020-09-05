import { deleteEntry } from '@/api'

export default {
  name: 'delete',
  display: {
    name: 'Delete',
    description: 'Delete this file',
    type: 'danger',
    icon: '#icon-delete'
  },
  supports: (entry) => entry.meta.can_write,
  handler: async (entry, { confirm, alert, loading }) => {
    try {
      await confirm({
        message: 'Delete this file?',
        confirmType: 'danger'
      })
    } catch { return }
    loading(true)
    try {
      await deleteEntry(entry.path)
      return { update: true }
    } catch (e) {
      alert(e.message)
    } finally {
      loading()
    }
  }
}
