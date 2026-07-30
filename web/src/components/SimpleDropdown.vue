<template>
  <div
    ref="rootEl"
    class="simple-dropdown"
    data-ui="dropdown"
    :class="{ active }"
    @focusout="onFocusOut"
  >
    <button
      ref="triggerEl"
      type="button"
      class="simple-dropdown__trigger"
      data-ui="dropdown-trigger"
      aria-haspopup="menu"
      :aria-label="ariaLabel"
      :aria-expanded="active"
      @click="triggerClicked"
      @keydown="onTriggerKeydown"
    >
      <slot />
    </button>
    <Teleport to="body">
      <Transition :name="dropdownTransition">
        <div
          v-show="active"
          ref="dropdownEl"
          class="simple-dropdown__dropdown glass-surface"
          data-ui="dropdown-panel"
          data-surface="glass"
          role="menu"
          :style="dropdownStyle"
          @keydown="onDropdownKeydown"
        >
          <ul
            class="simple-dropdown__menu"
            :style="menuMaxHeight ? { maxHeight: menuMaxHeight } : undefined"
          >
            <li
              v-for="item in items"
              :key="item.key"
              class="simple-dropdown__item"
              :class="{
                active: item.key === selectedKey,
                disabled: item.disabled,
              }"
              :role="hasSelection ? 'menuitemradio' : 'menuitem'"
              tabindex="-1"
              :aria-checked="hasSelection ? item.key === selectedKey : undefined"
              :aria-disabled="item.disabled || undefined"
              @click="selectItem(item)"
              @keydown.enter.prevent="selectItem(item)"
              @keydown.space.prevent="selectItem(item)"
            >
              <slot name="item" :item="item">{{ item.name }}</slot>
            </li>
          </ul>
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

interface SimpleDropdownItem {
  key: string
  name?: I18nText
  disabled?: boolean
}

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
  ariaLabel: {
    type: String,
  },
  items: {
    type: Array as PropType<SimpleDropdownItem[]>,
    default: () => [],
  },
  selectedKey: {
    type: String,
  },
  menuMaxHeight: {
    type: String,
  },
})
const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'select', key: string, item: SimpleDropdownItem): void
}>()

const active = ref(false)
const rootEl = ref<HTMLElement | null>(null)
const triggerEl = ref<HTMLElement | null>(null)
const dropdownEl = ref<HTMLElement | null>(null)
const dropdownStyle = ref<Record<string, string>>({})
const dropdownPlacedAbove = ref(false)
const focusMenuOnOpen = ref(false)
const hasSelection = computed(() => props.selectedKey !== undefined)
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
  focusMenuOnOpen.value = false
  setActive(!active.value)
}

const close = (focusTrigger = false) => {
  setActive(false)
  if (focusTrigger) triggerEl.value?.focus()
}

const focusInitialMenuItem = async () => {
  await nextTick()
  const items = getMenuItems()
  const selected = items.find(
    (item) => item.getAttribute('aria-checked') === 'true'
  )
  const itemToFocus = selected ?? items[0]
  itemToFocus?.focus()
  focusMenuOnOpen.value = false
}

const selectItem = (item: SimpleDropdownItem) => {
  if (item.disabled) return
  emit('select', item.key, item)
  close(true)
}

const onTriggerKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') {
    e.preventDefault()
    if (active.value) close(true)
    else {
      focusMenuOnOpen.value = true
      setActive(true)
    }
  } else if (e.key === 'ArrowDown') {
    e.preventDefault()
    if (active.value) void focusInitialMenuItem()
    else {
      focusMenuOnOpen.value = true
      setActive(true)
    }
  } else if (e.key === 'Escape' && active.value) {
    e.preventDefault()
    close(true)
  }
}

const onDropdownKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    e.preventDefault()
    close(true)
    return
  }

  const items = getMenuItems()
  if (!items.length) return
  const activeIndex = items.indexOf(document.activeElement as HTMLElement)
  let nextIndex: number | undefined
  if (e.key === 'ArrowDown') nextIndex = (activeIndex + 1) % items.length
  else if (e.key === 'ArrowUp') {
    nextIndex = (activeIndex - 1 + items.length) % items.length
  } else if (e.key === 'Home') nextIndex = 0
  else if (e.key === 'End') nextIndex = items.length - 1
  else if (e.key === 'Tab') {
    close()
    return
  }
  if (nextIndex !== undefined) {
    e.preventDefault()
    items[nextIndex].focus()
  }
}

const getMenuItems = () =>
  Array.from(
    dropdownEl.value?.querySelectorAll<HTMLElement>(
      '[role="menuitem"],[role="menuitemradio"],[role="menuitemcheckbox"]'
    ) ?? []
  ).filter((el) => el.getAttribute('aria-disabled') !== 'true')

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
  document[fn]('pointerdown', onDocumentPointerDown, true)
}

const updateDropdownPosition = () => {
  const trigger = triggerEl.value
  const dropdown = dropdownEl.value
  if (!trigger || !dropdown) return

  const triggerRect = trigger.getBoundingClientRect()
  const gap = 10
  const viewportPadding = 8
  const availableWidth = Math.max(
    window.innerWidth - viewportPadding * 2,
    0
  )
  const minWidth = Math.min(triggerRect.width, availableWidth)
  const dropdownWidth = Math.min(
    Math.max(dropdown.offsetWidth, minWidth),
    availableWidth
  )
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
    minWidth: `${Math.round(minWidth)}px`,
    maxWidth: `${Math.round(availableWidth)}px`,
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
    if (focusMenuOnOpen.value) await focusInitialMenuItem()
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
  padding: 0;
  border: 0;
  border-radius: 2px;
  background: transparent;
  color: inherit;
  font: inherit;
}

.simple-dropdown__dropdown {
  position: fixed;
  z-index: 1001;
  box-sizing: border-box;
  width: max-content;
  border: 1px solid var(--color-glass-border);
  border-radius: var(--radius-popover);
  outline: none;
  overflow: hidden;
  background-color: var(--color-bg-glass);
  box-shadow: var(--shadow-elevated);
}

.simple-dropdown__menu {
  margin: 0;
  padding: 0;
  width: 100%;
  overflow-y: auto;
}

.simple-dropdown__item {
  margin: 0;
  box-sizing: border-box;
  width: 100%;
  list-style-type: none;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 14px;

  &:hover,
  &:focus-visible {
    background-color: var(--color-bg-hover);
  }

  &:focus-visible {
    outline: 2px solid transparent;
    outline-offset: -2px;
    box-shadow: inset 0 0 0 2px var(--color-focus-ring);
  }

  &.active {
    background-color: var(--color-bg-selected);
  }

  &.disabled {
    cursor: not-allowed;
    opacity: 0.5;
  }

  &:first-child {
    border-radius:
      calc(var(--radius-popover) - 1px)
      calc(var(--radius-popover) - 1px)
      0
      0;
  }

  &:last-child {
    border-radius:
      0
      0
      calc(var(--radius-popover) - 1px)
      calc(var(--radius-popover) - 1px);
  }

  &:only-child {
    border-radius: calc(var(--radius-popover) - 1px);
  }
}
</style>
