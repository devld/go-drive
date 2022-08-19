import { useAppStore } from '@/store'
import { Entry } from '@/types'
import { addEntryIntoDataTransfer, DATA_TYPE_ENTRY } from '@/utils/entry'
import { Ref } from 'vue'
import { EntryEventData } from '.'

export type EntryDargAction = 'copy' | 'move' | 'link'

export interface EntryDragData {
  action: EntryDargAction
  from: Entry[]
  to: Entry | string
}

export type OnDragAction = (data: EntryDragData) => void

export const useEntryDarg = (
  enabled: Ref<boolean>,
  selectedEntry: Ref<Entry[]>,
  onDragAction: OnDragAction
) => {
  const store = useAppStore()
  let draggingEntries: Entry[] | undefined

  const onDragStart = ({ entry, event }: EntryEventData) => {
    if (!enabled.value) return
    const e = event as DragEvent
    const dt = e.dataTransfer!

    if (!entry) {
      // PathBar is not draggable
      e.preventDefault()
      return
    }

    let targets: Entry[]

    if (selectedEntry.value.length) {
      if (!selectedEntry.value.find((e) => e.path === entry.path)) {
        // if there are selected entries, but the dragging target is not selected
        e.preventDefault()
        return
      }
      targets = selectedEntry.value
    } else {
      targets = [entry]
    }

    draggingEntries = targets
    addEntryIntoDataTransfer(targets, dt)

    if (targets.some((e) => !e.meta.writable)) {
      dt.effectAllowed = store.isAdmin ? 'copyLink' : 'copy'
    } else {
      dt.effectAllowed = store.isAdmin ? 'all' : 'copyMove'
    }
  }

  const onDragOver = ({ entry, path, event }: EntryEventData) => {
    if (!enabled.value) return
    const e = event as DragEvent
    const dt = e.dataTransfer!

    if (entry && (entry.type !== 'dir' || !entry.meta.writable)) {
      dt.dropEffect = 'none'
      return
    }

    const toPath = entry ? entry.path : path!
    if (draggingEntries?.some((e) => e.path === toPath)) {
      dt.dropEffect = 'none'
      return
    }

    dt.dropEffect = 'move'
    if (e.ctrlKey) dt.dropEffect = 'copy'
    if (e.shiftKey) dt.dropEffect = 'link'

    e.preventDefault()
  }

  const onDrop = ({ entry, path, event }: EntryEventData) => {
    if (!enabled.value) return
    draggingEntries = undefined

    const e = event as DragEvent
    const dt = e.dataTransfer!

    let targets: Entry[]
    try {
      targets = JSON.parse(dt.getData(DATA_TYPE_ENTRY))
      if (!Array.isArray(targets)) return
    } catch {
      return
    }

    const toPath = entry ? entry.path : path!

    if (targets.some((e) => e.path === toPath)) return

    e.preventDefault()

    let action: EntryDargAction = 'move'
    if (e.ctrlKey) action = 'copy'
    if (e.shiftKey) action = 'link'

    onDragAction({
      action,
      from: targets,
      to: entry ?? path!,
    })
  }

  return { onDragStart, onDragOver, onDrop }
}
