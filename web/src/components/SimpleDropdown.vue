<template>
  <div
    ref="rootEl"
    class="simple-dropdown"
    data-ui="dropdown"
    :class="{ active }"
    @focusout="onFocusOut"
  >
    <span
      ref="triggerEl"
      class="simple-dropdown__trigger"
      data-ui="dropdown-trigger"
      role="button"
      tabindex="0"
      aria-haspopup="true"
      :aria-expanded="active"
      @click="triggerClicked"
      @keydown="onTriggerKeydown"
    >
      <slot />
    </span>
    <Teleport to="body">
      <Transition :name="dropdownTransition">
        <div
          v-show="active"
          ref="dropdownEl"
          class="simple-dropdown__dropdown glass-surface"
          data-ui="dropdown-panel"
          data-surface="glass"
          :style="dropdownStyle"
          @keydown="onDropdownKeydown"
        >
          <slot name="dropdown" />
        </div>
      </Transition>
    </Teleport>
  </div>
</template>
<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  ref,
  watch,
  watchEffect,
  PropType,
} from 'vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
  },
  transition: {
    type: String,
    default: 'fade-slide-down',
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
const dropdownEl = ref<HTMLElement | null>(null)
const dropdownStyle = ref<Record<string, string>>({})
const dropdownPlacedAbove = ref(false)
const dropdownTransition = computed(() =>
  props.transition === 'fade-slide-down' && dropdownPlacedAbove.value
    ? 'fade-slide-up'
    : props.transition
)

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
  if (
    next &&
    (root.contains(next) || dropdownEl.value?.contains(next))
  ) {
    return
  }
  setActive(false)
}

// Close on pointer interactions outside the component.
const onDocumentPointerDown = (e: Event) => {
  const root = rootEl.value
  if (
    e.target instanceof Node &&
    (root?.contains(e.target) || dropdownEl.value?.contains(e.target))
  ) {
    return
  }
  setActive(false)
}

const bindOutside = (bind: boolean) => {
  const fn = bind ? 'addEventListener' : 'removeEventListener'
  document[fn]('mousedown', onDocumentPointerDown, true)
  document[fn]('touchstart', onDocumentPointerDown, true)
}

const updateDropdownPosition = () => {
  const trigger = triggerEl.value
  const dropdown = dropdownEl.value
  if (!trigger || !dropdown) return

  const triggerRect = trigger.getBoundingClientRect()
  const gap = 10
  const viewportPadding = 8
  const dropdownWidth = dropdown.offsetWidth
  const dropdownHeight = dropdown.offsetHeight

  let left =
    props.position === 'bottom-left'
      ? triggerRect.right - dropdownWidth
      : triggerRect.left
  left = Math.min(
    Math.max(left, viewportPadding),
    window.innerWidth - dropdownWidth - viewportPadding
  )

  let top = triggerRect.bottom + gap
  const topWhenFlipped = triggerRect.top - dropdownHeight - gap
  if (
    top + dropdownHeight > window.innerHeight - viewportPadding &&
    topWhenFlipped >= viewportPadding
  ) {
    top = topWhenFlipped
    dropdownPlacedAbove.value = true
  } else {
    dropdownPlacedAbove.value = false
  }

  dropdownStyle.value = {
    left: `${Math.round(left)}px`,
    top: `${Math.round(top)}px`,
  }
}

const bindPositionUpdates = (bind: boolean) => {
  const fn = bind ? 'addEventListener' : 'removeEventListener'
  window[fn]('resize', updateDropdownPosition)
  document[fn]('scroll', updateDropdownPosition, true)
}

watch(active, async (v) => {
  bindOutside(v)
  bindPositionUpdates(v)
  if (v) {
    await nextTick()
    updateDropdownPosition()
  }
})

onBeforeUnmount(() => {
  bindOutside(false)
  bindPositionUpdates(false)
})
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
  position: fixed;
  z-index: 1001;
  border-radius: var(--radius-popover);
  overflow: hidden;
  background-color: var(--color-bg-glass);
  box-shadow: var(--shadow-elevated);
}
</style>
