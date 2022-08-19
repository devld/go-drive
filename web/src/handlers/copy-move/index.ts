import { T } from '@/i18n'
import { copyOrMove } from '@/utils/entry'
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
    handler: ({ entry: entries }, { open }) => {
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
            const executed = await copyOrMove(isMove, entries, path)
            resolve({ update: executed.length > 0 })
          },
        })
      })
    },
  }
}

export const copy = createHandler(false)
export const move = createHandler(true)
