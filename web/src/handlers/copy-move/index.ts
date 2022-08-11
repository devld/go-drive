import { copyEntry, deleteTask, moveEntry } from '@/api'
import { T } from '@/i18n'
import { Entry, Task } from '@/types'
import {
  formatBytes,
  pathClean,
  pathJoin,
  taskDone,
  TASK_CANCELLED,
} from '@/utils'
import { EntryHandler } from '../types'

const createHandler = (isMove: boolean): EntryHandler => {
  return {
    name: isMove ? 'move' : 'copy',
    display: {
      name: T(
        isMove ? 'handler.copy_move.move_to' : 'handler.copy_move.copy_to'
      ),
      description: T(
        isMove ? 'handler.copy_move.move_desc' : 'handler.copy_move.copy_desc'
      ),
      icon: isMove ? '#icon-move' : '#icon-copy',
    },
    multiple: true,
    supports: isMove
      ? ({ entry, parent }) =>
          !!(entry.every((e) => e.meta.writable) && parent?.meta.writable)
      : () => true,
    handler: ({ entry: entries }, { confirm, alert, loading, open }) => {
      return new Promise((resolve) => {
        open({
          title: T(
            isMove
              ? 'handler.copy_move.move_open_title'
              : 'handler.copy_move.copy_open_title'
          ),
          type: 'dir',
          filter: 'write',
          async onOk(path) {
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
                const dest = pathClean(pathJoin(path, entry.name))
                loading({
                  text: T(
                    isMove
                      ? 'handler.copy_move.moving'
                      : 'handler.copy_move.copying',
                    { n: entry.name }
                  ),
                })
                const copyOrMove = isMove ? moveEntry : copyEntry
                await taskDone(copyOrMove(entry.path, dest, override), (t) => {
                  if (canceled) return false
                  task = t
                  loading({
                    text: T(
                      isMove
                        ? 'handler.copy_move.moving'
                        : 'handler.copy_move.copying',
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
              }
              resolve({ update: true })
            } catch (e: any) {
              if (e === TASK_CANCELLED) return
              alert(e.message)
              throw e
            } finally {
              loading()
            }
          },
        })
      })
    },
  }
}

export const copy = createHandler(false)
export const move = createHandler(true)