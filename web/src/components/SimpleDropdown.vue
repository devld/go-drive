<template>
  <div
    ref="rootEl"
    class="simple-dropdown"
    :class="{ active }"
    @focusout="onFocusOut"
  >
    <span
      ref="triggerEl"
      class="simple-dropdown__trigger"
      role="button"
      tabindex="0"
      aria-haspopup="true"
      :aria-expanded="active"
      @click="triggerClicked"
      @keydown="onTriggerKeydown"
    >
      <slot />
    </span>
    <Transition :name="transition">
      <div
        v-show="active"
        class="simple-dropdown__dropdown"
        :class="`simple-dropdown__dropdown--${position}`"
        @keydown="onDropdownKeydown"
      >
        <slot name="dropdown" />
      </div>
    </Transition>
  </div>
</template>
<script setup lang="ts">
import { onBeforeUnmount, ref, watch, watchEffect, PropType } from 'vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
  },
  transition: {
    type: String,
    default: 'top-fade',
  },
  position: {
    type: String as PropType<'bottom-left' | 'bottom-right'>,
    default: 'bottom-left',
  },
})
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const active = ref(false)
const rootEl = ref<HTMLElement | null>(null)
const triggerEl = ref<HTMLElement | null>(null)

watchEffect(() => {
  active.value = !!props.modelValue
})

const setActive = (v: boolean) => {
  if (active.value === v) return
  active.value = v
  emit('update:modelValue', v)
}

const triggerClicked = () => {
  setActive(!active.value)
}

const close = (focusTrigger = false) => {
  setActive(false)
  if (focusTrigger) triggerEl.value?.focus()
}

const onTriggerKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') {
    e.preventDefault()
    setActive(!active.value)
  } else if (e.key === 'ArrowDown') {
    e.preventDefault()
    setActive(true)
  } else if (e.key === 'Escape' && active.value) {
    e.preventDefault()
    close(true)
  }
}

const onDropdownKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    e.preventDefault()
    close(true)
  }
}

// Close when focus moves outside the component (e.g. keyboard Tab).
// Clicking a non-focusable item keeps focus on the trigger, so this does
// not interfere with selecting such items.
const onFocusOut = (e: FocusEvent) => {
  const root = rootEl.value
  if (!root) return
  const next = e.relatedTarget as Node | null
  if (next && root.contains(next)) return
  setActive(false)
}

// Close on pointer interactions outside the component.
const onDocumentPointerDown = (e: Event) => {
  const root = rootEl.value
  if (root && e.target instanceof Node && root.contains(e.target)) return
  setActive(false)
}

const bindOutside = (bind: boolean) => {
  const fn = bind ? 'addEventListener' : 'removeEventListener'
  document[fn]('mousedown', onDocumentPointerDown, true)
  document[fn]('touchstart', onDocumentPointerDown, true)
}

watch(active, (v) => bindOutside(v))

onBeforeUnmount(() => bindOutside(false))
</script>
<style lang="scss">
.simple-dropdown {
  display: inline-block;
  position: relative;
}

.simple-dropdown__trigger {
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  border-radius: 2px;
}

.simple-dropdown__dropdown {
  position: absolute;
  margin-top: 10px;
  z-index: 999;
  background-color: var(--secondary-bg-color);
  box-shadow: var(--dialog-content-shadow);
}

.simple-dropdown__dropdown--bottom-left {
  right: 0;
  left: unset;
}

.simple-dropdown__dropdown--bottom-right {
  right: unset;
  left: 0;
}
</style>
