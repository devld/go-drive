<template>
  <div
    v-if="eager || overlayShowing"
    v-show="overlayShowing"
    ref="overlayEl"
    class="dialog-view dialog-view__overlay"
    @click="overlayClicked"
  >
    <Transition :name="transition" @after-leave="onDialogClosed">
      <div
        v-if="eager || contentShowing"
        v-show="contentShowing"
        class="dialog-view__content"
      >
        <div v-if="$slots.header || title" class="dialog-view__header">
          <slot name="header">
            <span>{{ title }}</span>
          </slot>
          <button
            v-if="closeable"
            class="dialog-view__close-button plain-button"
            @click="closeButtonClicked"
          >
            <Icon svg="#icon-close" />
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
import { nextTick, onBeforeUnmount, ref, watchEffect } from 'vue'
import { addScrollLockedCount, getScrollLockedCount } from './state'

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
  lockScroll: {
    type: Boolean,
  },
})

const emit = defineEmits<{
  (e: 'closed'): void
  (e: 'update:show', show: boolean): void
}>()

const overlayShowing = ref(false)
const contentShowing = ref(false)
const overlayEl = ref(null)
let scrollLocked = false

const overlayClicked = (e: MouseEvent) => {
  if (!props.overlayClose) return
  if (props.closeable && e.target === overlayEl.value) {
    close()
  }
}
const close = () => emit('update:show', false)
const closeButtonClicked = () => close()

const onKeyDown = (e: KeyboardEvent) => {
  if (!props.escClose) return
  if (props.closeable && e.key === 'Escape' && props.show) {
    close()
    e.preventDefault()
  }
}

const onDialogClosed = () => {
  overlayShowing.value = false
  emit('closed')
}

const setupEvents = () => {
  window.addEventListener('keydown', onKeyDown)
}

const removeEvents = () => {
  window.removeEventListener('keydown', onKeyDown)
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

watchEffect(() => {
  const val = props.show
  if (val) {
    overlayShowing.value = val
    nextTick(() => {
      contentShowing.value = true
    })
    setupEvents()
  } else {
    contentShowing.value = false
    if (props.transition === 'none') {
      overlayShowing.value = false
    }
    removeEvents()
  }
})

watchEffect(() => {
  onDialogVisibleChanged(overlayShowing.value)
})

onBeforeUnmount(() => {
  onDialogVisibleChanged(false)
  removeEvents()
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
  bottom: 0;
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
</style>
