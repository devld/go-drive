<template>
  <div
    v-if="eager || overlayShowing"
    v-show="overlayShowing"
    ref="overlayEl"
    class="dialog-view dialog-view__overlay"
    :class="{ 'dialog-view--fullscreen': fullscreen }"
    @click="overlayClicked"
  >
    <Transition
      :name="transition"
      @after-enter="focusInitial"
      @after-leave="onDialogClosed"
    >
      <div
        v-if="eager || contentShowing"
        v-show="contentShowing"
        ref="contentEl"
        class="dialog-view__content"
        role="dialog"
        aria-modal="true"
        :aria-label="ariaLabel"
        tabindex="-1"
        @keydown="onContentKeyDown"
      >
        <div v-if="$slots.header || title" class="dialog-view__header">
          <slot name="header">
            <span>{{ title }}</span>
          </slot>
          <button
            v-if="closeable"
            type="button"
            class="dialog-view__close-button plain-button"
            :title="closeLabel"
            :aria-label="closeLabel"
            @click="closeButtonClicked"
          >
            <Icon name="close" aria-hidden="true" />
          </button>
        </div>
        <div class="dialog-view__body">
          <slot />
        </div>
        <div v-if="$slots.footer" class="dialog-view__footer">
          <slot name="footer" />
        </div>
      </div>
    </Transition>
  </div>
</template>
<script lang="ts">
export default { name: 'DialogView' }
</script>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { s, T } from '@/i18n'
import {
  addScrollLockedCount,
  getScrollLockedCount,
  isTopDialog,
  pushDialog,
  removeDialog,
  type DialogController,
} from './state'

const props = defineProps({
  show: {
    type: Boolean,
  },
  title: {
    type: [String, Object] as PropType<I18nText>,
  },
  transition: {
    type: String,
    default: 'fade',
  },
  overlayClose: {
    type: Boolean,
  },
  escClose: {
    type: Boolean,
  },
  closeable: {
    type: Boolean,
    default: true,
  },
  eager: {
    type: Boolean,
  },
  fullscreen: {
    type: Boolean,
  },
  lockScroll: {
    type: Boolean,
    default: true,
  },
})

const emit = defineEmits<{
  (e: 'closed'): void
  (e: 'update:show', show: boolean): void
}>()

const overlayShowing = ref(false)
const contentShowing = ref(false)
const overlayEl = ref(null)
const contentEl = ref<HTMLElement | null>(null)
let scrollLocked = false
let lastFocused: HTMLElement | null = null

const ariaLabel = computed(() => (props.title ? s(props.title) : undefined))
const closeLabel = computed(() => s(T('dialog.base.close')))

const overlayClicked = (e: MouseEvent) => {
  if (!props.overlayClose) return
  if (props.closeable && e.target === overlayEl.value) {
    close()
  }
}
const close = () => emit('update:show', false)
const closeButtonClicked = () => close()

const dialogController: DialogController = {
  canEscClose: () => !!props.escClose && !!props.closeable && !!props.show,
  requestClose: () => close(),
}

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'textarea:not([disabled])',
  'select:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

const getFocusable = (root: HTMLElement | null): HTMLElement[] => {
  if (!root) return []
  return Array.from(
    root.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)
  ).filter((el) => el.offsetParent !== null || el === document.activeElement)
}

const focusInitial = () => {
  const content = contentEl.value
  if (!content) return
  const active = document.activeElement as HTMLElement | null
  // Respect focus that inner content already claimed (e.g. an input field).
  if (active && active !== content && content.contains(active)) return

  const autofocus = content.querySelector<HTMLElement>('[autofocus]')
  if (autofocus) return autofocus.focus()

  const field = content.querySelector<HTMLElement>(
    'input:not([disabled]),textarea:not([disabled]),select:not([disabled])'
  )
  if (field) return field.focus()

  // Default to the primary (last) footer button so Enter/Esc work right away.
  const buttons = content.querySelectorAll<HTMLElement>(
    '.dialog-view__footer button:not([disabled])'
  )
  if (buttons.length) return buttons[0].focus()

  content.focus()
}

// Focusing during the frame the dialog opens in does not stick: focus() lands
// but the v-if/v-show + <Transition> enter work runs in that same frame and
// resets activeElement back to <body>. Waiting until the next frame (a
// microtask via nextTick is still too early) makes the focus stick. The
// transition's @after-enter re-asserts it as a final guarantee.
const focusInitialDeferred = () => {
  requestAnimationFrame(() => requestAnimationFrame(() => focusInitial()))
}

const restoreFocus = () => {
  const el = lastFocused
  lastFocused = null
  if (el && typeof el.focus === 'function' && document.contains(el)) {
    el.focus()
  }
}

// Keep Tab focus within the topmost dialog (basic focus trap).
const onContentKeyDown = (e: KeyboardEvent) => {
  if (e.key !== 'Tab') return
  if (!isTopDialog(dialogController)) return
  const content = contentEl.value
  if (!content) return
  const focusable = getFocusable(content)
  const active = document.activeElement as HTMLElement | null
  if (!focusable.length) {
    e.preventDefault()
    content.focus()
    return
  }
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (e.shiftKey) {
    if (active === first || !content.contains(active)) {
      e.preventDefault()
      last.focus()
    }
  } else if (active === last || !content.contains(active)) {
    e.preventDefault()
    first.focus()
  }
}

const onDialogClosed = () => {
  overlayShowing.value = false
  restoreFocus()
  emit('closed')
}

const onDialogVisibleChanged = (showing: boolean) => {
  if (!props.lockScroll) return
  if (showing) {
    if (scrollLocked) return
    addScrollLockedCount(1)
    scrollLocked = true
  } else {
    if (!scrollLocked) return
    addScrollLockedCount(-1)
    scrollLocked = false
  }
  if (getScrollLockedCount() > 0) {
    document.body.classList.add('dialog-view--scrollable-locked')
  } else {
    document.body.classList.remove('dialog-view--scrollable-locked')
  }
}

watch(
  () => props.show,
  (val) => {
    if (val) {
      lastFocused = document.activeElement as HTMLElement | null
      overlayShowing.value = true
      nextTick(() => {
        contentShowing.value = true
        focusInitialDeferred()
      })
    } else {
      contentShowing.value = false
      if (props.transition === 'none') {
        overlayShowing.value = false
        restoreFocus()
      }
    }
  },
  { immediate: true }
)

watch(
  overlayShowing,
  (showing) => {
    onDialogVisibleChanged(showing)
    if (showing) pushDialog(dialogController)
    else removeDialog(dialogController)
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  onDialogVisibleChanged(false)
  removeDialog(dialogController)
})
</script>
<style lang="scss">
.dialog-view--scrollable-locked {
  overflow: hidden;
}

.dialog-view__overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 100%;
  min-height: -webkit-fill-available;
  max-height: -webkit-fill-available;
  overflow: hidden;
  background-color: var(--dialog-overlay-bg-color);
  z-index: 1000;

  display: flex;
  justify-content: center;
  align-items: center;
}

.dialog-view__content {
  background-color: var(--secondary-bg-color);
  box-shadow: var(--dialog-content-shadow);
}

.dialog-view__header {
  position: relative;
  min-width: 200px;
  font-size: 28px;
  font-weight: normal;
  -webkit-user-select: none;
  user-select: none;
  padding: 16px 48px 16px 16px;
}

.dialog-view__close-button {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  right: 12px;
  white-space: nowrap;
  overflow: hidden;
}

.dialog-view__body {
  overflow: hidden;
}

.dialog-view--fullscreen {
  .dialog-view__content {
    width: 100%;
    height: 100%;
    overflow: hidden;

    display: flex;
    flex-direction: column;
  }

  .dialog-view__body {
    width: 100%;
    flex: 1;

    display: flex;
    justify-content: center;
    align-items: center;
  }
}
</style>
