/** Adapt Milkdown's pointer-driven block and table controls to touch taps. */
export function installTouchControls(
  root: HTMLElement,
  isReadonly: () => boolean
) {
  function onPointerDown(event: PointerEvent) {
    if (
      isReadonly() ||
      event.pointerType !== 'touch' ||
      !(event.target instanceof Element)
    ) {
      return
    }

    const target = event.target
    const table = target.closest('.milkdown-table-block')
    for (const other of Array.from(
      root.querySelectorAll('.milkdown-table-block')
    )) {
      if (other === table) continue
      for (const control of Array.from(
        other.querySelectorAll<HTMLElement>('.handle, .handle .button-group')
      )) {
        control.dataset.show = 'false'
      }
    }

    if (
      !target.closest('.ProseMirror') ||
      target.closest(
        [
          '.milkdown-block-handle',
          '.milkdown-table-block .handle',
          '.milkdown-slash-menu',
          '.milkdown-code-block .tools',
          '.milkdown-code-block .list-wrapper',
        ].join(', ')
      )
    ) {
      return
    }

    // Reuse Milkdown's control positioning for a tap without pointer movement.
    target.dispatchEvent(
      new PointerEvent('pointermove', {
        bubbles: true,
        clientX: event.clientX,
        clientY: event.clientY,
        pointerType: 'touch',
      })
    )
  }

  function onPointerLeave(event: PointerEvent) {
    if (
      isReadonly() ||
      event.pointerType !== 'touch' ||
      !(event.target instanceof Element)
    ) {
      return
    }

    const table = event.target.closest('.milkdown-table-block')
    if (table?.querySelector('.handle[data-show="true"]')) {
      // Keep the handles available for the next tap; tapping elsewhere clears them.
      event.stopPropagation()
    }
  }

  root.addEventListener('pointerdown', onPointerDown)
  root.addEventListener('pointerleave', onPointerLeave, true)
  return () => {
    root.removeEventListener('pointerdown', onPointerDown)
    root.removeEventListener('pointerleave', onPointerLeave, true)
  }
}
