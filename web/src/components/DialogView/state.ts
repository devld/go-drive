let scrollLockedCount = 0

export function addScrollLockedCount(delta: number) {
  return (scrollLockedCount += delta)
}

export function getScrollLockedCount() {
  return scrollLockedCount
}

export interface DialogController {
  /** Whether this dialog may be closed by the Escape key right now. */
  canEscClose: () => boolean
  /** Request to close this dialog. */
  requestClose: () => void
}

// Stack of currently visible dialogs, ordered by open time (last is topmost).
const dialogStack: DialogController[] = []
let escListenerBound = false

function onGlobalKeyDown(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  const top = dialogStack[dialogStack.length - 1]
  if (!top) return
  // Only the topmost dialog reacts to Escape, so stacked dialogs close one by
  // one instead of all at once. When the topmost dialog opts out of
  // Escape-to-close we let the event keep propagating, so other handlers (e.g.
  // a preview that closes itself on Escape) still work.
  console.log('top.canEscClose()', top.canEscClose())
  if (!top.canEscClose()) return
  e.preventDefault()
  e.stopPropagation()
  top.requestClose()
}

export function pushDialog(controller: DialogController) {
  if (dialogStack.includes(controller)) return
  dialogStack.push(controller)
  if (!escListenerBound) {
    window.addEventListener('keydown', onGlobalKeyDown, true)
    escListenerBound = true
  }
}

export function removeDialog(controller: DialogController) {
  const i = dialogStack.lastIndexOf(controller)
  if (i !== -1) dialogStack.splice(i, 1)
}

export function isTopDialog(controller: DialogController) {
  return dialogStack[dialogStack.length - 1] === controller
}
