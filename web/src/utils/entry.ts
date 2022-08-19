import { copyEntry, deleteTask, fileUrl, moveEntry } from '@/api'
import { T } from '@/i18n'
import { Entry, Task } from '@/types'
import { formatBytes, pathClean, pathJoin, taskDone, TASK_CANCELLED } from '.'
import { alert, confirm, loading } from './ui-utils'

export const DATA_TYPE_ENTRY = 'application/go-drive-entry'

export const addEntryIntoDataTransfer = (
  entry: Entry | Entry[],
  dt: DataTransfer
) => {
  const entries = Array.isArray(entry) ? entry : [entry]

  dt.setData(DATA_TYPE_ENTRY, JSON.stringify(entries))

  const itemsURI: string[] = []
  for (const entry of entries) {
    let temp
    if (entry && entry.type === 'file') {
      temp = fileUrl(entry.path, entry.meta)
    }
    if (temp) itemsURI.push(temp)
  }

  if (itemsURI.length > 0) {
    const uris = itemsURI.join('\r\n')
    dt.setData('text/plain', uris)
    dt.setData('text/uri-list', uris)
  }
}

export const copyOrMove = async (
  isMove: boolean,
  entries: Entry[],
  toDir: string
) => {
  const executedEntries: Entry[] = []

  let override = true
  try {
    await confirm({
      message: T('handler.copy_move.override_or_skip'),
      confirmType: 'danger',
      confirmText: T('handler.copy_move.override'),
      cancelText: T('handler.copy_move.skip'),
    })
  } catch {
    override = false
  }
  let canceled = false
  let task: Task<Entry>
  const onCancel = () => {
    canceled = true
    return task && deleteTask(task.id)
  }
  try {
    for (const i in entries) {
      if (canceled) break
      const entry = entries[i]
      const dest = pathClean(pathJoin(toDir, entry.name))
      loading({
        text: T(
          isMove ? 'handler.copy_move.moving' : 'handler.copy_move.copying',
          { n: entry.name }
        ),
      })
      const copyOrMove = isMove ? moveEntry : copyEntry
      await taskDone(copyOrMove(entry.path, dest, override), (t) => {
        if (canceled) return false
        task = t
        loading({
          text: T(
            isMove ? 'handler.copy_move.moving' : 'handler.copy_move.copying',
            {
              n: entry.name,
              p: task.progress
                ? `${formatBytes(task.progress.loaded)}/${formatBytes(
                    task.progress.total
                  )}`
                : '',
            }
          ),
          onCancel,
        })
      })
      executedEntries.push(entry)
    }
    return executedEntries
  } catch (e: any) {
    if (e === TASK_CANCELLED) return executedEntries
    alert(e.message)
    throw e
  } finally {
    loading()
  }
}
