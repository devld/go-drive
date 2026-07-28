import { useAppStore } from '@/store'
import { Entry } from '@/types'
import { isParentPath, pathClean, pathJoin } from '@/utils'
import { addEntryIntoDataTransfer, DATA_TYPE_ENTRY } from '@/utils/entry'
import { isPrimaryModifierPressed } from '@/utils/platform'
import { Ref, ref } from 'vue'
import { EntryEventData } from '.'

export type EntryDragAction = 'copy' | 'move' | 'link'
export type EntryDropInvalidReason =
  | 'not-directory'
  | 'target-readonly'
  | 'same-target'
  | 'descendant'

export interface EntryDragData {
  action: EntryDragAction
  from: Entry[]
  to: Entry | string
}

export interface EntryDragState {
  active: boolean
  entries: Entry[]
  targetPath?: string
  action?: EntryDragAction
  allowed: boolean
  invalidReason?: EntryDropInvalidReason
}

export interface EntryDropIntent {
  action: EntryDragAction
  targetPath: string
  allowed: boolean
  invalidReason?: EntryDropInvalidReason
}

export interface EntryDragModifiers {
  copy: boolean
  link: boolean
}

export type OnDragAction = (data: EntryDragData) => void
export type GetDragElements = (entries: Entry[]) => HTMLElement[]

export const getEntryDropTargetClass = (
  state: EntryDragState | undefined,
  path: string
) => {
  const isTarget = state?.targetPath === path
  return {
    'entry-link--drop-valid': !!isTarget && state.allowed,
    'entry-link--drop-invalid': !!isTarget && !state.allowed,
    'entry-link--drop-move':
      !!isTarget && state.allowed && state.action === 'move',
    'entry-link--drop-copy':
      !!isTarget && state.allowed && state.action === 'copy',
    'entry-link--drop-link':
      !!isTarget && state.allowed && state.action === 'link',
  }
}

const emptyDragState = (): EntryDragState => ({
  active: false,
  entries: [],
  allowed: false,
})

const getModifiers = (event: DragEvent): EntryDragModifiers => ({
  copy: isPrimaryModifierPressed(event),
  link: event.shiftKey,
})

const hasEntryData = (dt: DataTransfer) =>
  Array.from(dt.types).includes(DATA_TYPE_ENTRY)

const getDraggedEntries = (dt: DataTransfer): Entry[] | undefined => {
  try {
    const entries = JSON.parse(dt.getData(DATA_TYPE_ENTRY))
    if (!Array.isArray(entries)) return
    return entries
  } catch {
    return
  }
}

export const resolveEntryDropIntent = ({
  entries,
  target,
  modifiers,
  isAdmin,
  sourceDirWritable,
}: {
  entries: Entry[]
  target: Entry | string
  modifiers: EntryDragModifiers
  isAdmin: boolean
  sourceDirWritable: boolean
}): EntryDropIntent => {
  const targetPath = typeof target === 'string' ? target : target.path
  const canMove =
    sourceDirWritable && entries.every((entry) => entry.meta.writable)
  let action: EntryDragAction = canMove ? 'move' : 'copy'

  if (modifiers.copy) action = 'copy'
  if (modifiers.link && isAdmin) action = 'link'

  if (
    typeof target !== 'string' &&
    (target.type !== 'dir' || !target.meta.writable)
  ) {
    return {
      action,
      targetPath,
      allowed: false,
      invalidReason:
        target.type !== 'dir' ? 'not-directory' : 'target-readonly',
    }
  }

  for (const entry of entries) {
    const sourcePath = pathClean(entry.path)
    const destinationPath = pathClean(pathJoin(targetPath, entry.name))

    if (destinationPath === sourcePath || targetPath === sourcePath) {
      return {
        action,
        targetPath,
        allowed: false,
        invalidReason: 'same-target',
      }
    }

    if (entry.type === 'dir' && isParentPath(targetPath, sourcePath)) {
      return {
        action,
        targetPath,
        allowed: false,
        invalidReason: 'descendant',
      }
    }
  }

  return { action, targetPath, allowed: true }
}

export const useEntryDrag = (
  enabled: Ref<boolean>,
  selectedEntries: Ref<Entry[]>,
  currentEntry: Ref<Entry | undefined>,
  onDragAction: OnDragAction,
  onSelectionChange: (entries: Entry[]) => void,
  getDragElements: GetDragElements
) => {
  const store = useAppStore()
  const dragState = ref<EntryDragState>(emptyDragState())

  const reset = () => {
    dragState.value = emptyDragState()
  }

  const onDragStart = ({ entry, event }: EntryEventData) => {
    if (!enabled.value) return
    const e = event as DragEvent
    const dt = e.dataTransfer
    if (!dt) return

    if (!entry || entry.name === '..') {
      e.preventDefault()
      return
    }

    let targets: Entry[]
    if (selectedEntries.value.some((item) => item.path === entry.path)) {
      targets = [...selectedEntries.value]
    } else {
      targets = [entry]
      selectedEntries.value = targets
      onSelectionChange(targets)
    }

    addEntryIntoDataTransfer(targets, dt)

    const sourceList = (e.currentTarget as HTMLElement | null)?.closest(
      '.entry-list'
    )
    const dragImage = document.createElement('div')
    dragImage.className = [
      'entry-drag-image',
      sourceList?.className ?? '',
    ].join(' ')
    const dragStack = document.createElement('div')
    dragStack.className = 'entry-drag-image__stack'
    dragImage.appendChild(dragStack)
    const visibleElements = getDragElements(targets).slice(0, 3)
    if (visibleElements.length > 0) {
      const rects = visibleElements.map((element) =>
        element.getBoundingClientRect()
      )
      const width = Math.max(...rects.map((rect) => rect.width))
      const height = Math.max(...rects.map((rect) => rect.height))

      visibleElements.forEach((element, index) => {
        const layer = document.createElement('div')
        layer.className = 'entry-drag-image__layer'
        layer.style.left = `${index * 7}px`
        layer.style.top = `${index * 7}px`
        layer.style.zIndex = `${visibleElements.length - index}`
        layer.style.width = `${width}px`
        layer.style.height = `${height}px`

        const item = element.cloneNode(true) as HTMLElement
        item.classList.remove('selected', 'dragging')
        item.classList.add('entry-drag-image__item')
        item.style.width = `${width}px`
        item.style.height = `${height}px`
        layer.appendChild(item)
        dragStack.appendChild(layer)
      })

      const sourceItem = (
        e.currentTarget as HTMLElement | null
      )?.closest('.entry-list__item')
      const sourceRect = sourceItem?.getBoundingClientRect() ?? rects[0]
      const pointerX = Math.max(
        0,
        Math.min(sourceRect.width, e.clientX - sourceRect.left)
      )
      const pointerY = Math.max(
        0,
        Math.min(sourceRect.height, e.clientY - sourceRect.top)
      )

      const shadowPadding = 16
      const stackWidth = width + (visibleElements.length - 1) * 7
      const stackHeight = height + (visibleElements.length - 1) * 7
      dragStack.style.left = `${shadowPadding}px`
      dragStack.style.top = `${shadowPadding}px`
      dragStack.style.width = `${stackWidth}px`
      dragStack.style.height = `${stackHeight}px`
      dragImage.style.width = `${stackWidth + shadowPadding * 2}px`
      dragImage.style.height = `${stackHeight + shadowPadding * 2}px`
      dragImage.style.left = `${sourceRect.left - shadowPadding}px`
      dragImage.style.top = `${sourceRect.top - shadowPadding}px`
      document.body.appendChild(dragImage)
      dt.setDragImage(
        dragImage,
        pointerX + shadowPadding,
        pointerY + shadowPadding
      )
      requestAnimationFrame(() => dragImage.remove())
    }

    const canMove =
      !!currentEntry.value?.meta.writable &&
      targets.every((item) => item.meta.writable)
    if (canMove) {
      dt.effectAllowed = store.isAdmin ? 'all' : 'copyMove'
    } else {
      dt.effectAllowed = store.isAdmin ? 'copyLink' : 'copy'
    }

    dragState.value = {
      active: true,
      entries: targets,
      action: canMove ? 'move' : 'copy',
      allowed: false,
    }
  }

  const onDragOver = ({ entry, path, event }: EntryEventData) => {
    if (!enabled.value) return
    const e = event as DragEvent
    const dt = e.dataTransfer
    if (!dt || !hasEntryData(dt)) return

    const target = entry ?? path
    if (target === undefined) return

    const intent = resolveEntryDropIntent({
      entries: dragState.value.entries,
      target,
      modifiers: getModifiers(e),
      isAdmin: store.isAdmin,
      sourceDirWritable: !!currentEntry.value?.meta.writable,
    })

    dragState.value = {
      ...dragState.value,
      active: true,
      targetPath: intent.targetPath,
      action: intent.action,
      allowed: intent.allowed,
      invalidReason: intent.invalidReason,
    }

    dt.dropEffect = intent.allowed ? intent.action : 'none'
    e.preventDefault()
  }

  const onDragLeave = ({ entry, path, event }: EntryEventData) => {
    const targetPath = entry?.path ?? path
    if (targetPath !== dragState.value.targetPath) return

    const e = event as DragEvent
    const currentTarget = e.currentTarget
    const relatedTarget = e.relatedTarget
    if (
      currentTarget instanceof HTMLElement &&
      relatedTarget instanceof Node &&
      currentTarget.contains(relatedTarget)
    ) {
      return
    }

    const canMove =
      !!currentEntry.value?.meta.writable &&
      dragState.value.entries.every((item) => item.meta.writable)
    dragState.value = {
      ...dragState.value,
      targetPath: undefined,
      action: canMove ? 'move' : 'copy',
      allowed: false,
      invalidReason: undefined,
    }
  }

  const onDrop = ({ entry, path, event }: EntryEventData) => {
    if (!enabled.value) return
    const e = event as DragEvent
    const dt = e.dataTransfer
    if (!dt || !hasEntryData(dt)) return

    e.preventDefault()
    e.stopPropagation()

    const targets = getDraggedEntries(dt)
    const target = entry ?? path
    if (!targets || target === undefined) {
      reset()
      return
    }

    const intent = resolveEntryDropIntent({
      entries: targets,
      target,
      modifiers: getModifiers(e),
      isAdmin: store.isAdmin,
      sourceDirWritable: !!currentEntry.value?.meta.writable,
    })
    reset()
    if (!intent.allowed) return

    onDragAction({
      action: intent.action,
      from: targets,
      to: target,
    })
  }

  const onDragEnd = () => reset()

  return {
    dragState,
    onDragStart,
    onDragOver,
    onDragLeave,
    onDrop,
    onDragEnd,
  }
}
