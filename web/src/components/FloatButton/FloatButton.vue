<template>
  <div
    class="float-button"
    :class="[
      `float-button__posi-${position}`,
      modelValue ? 'float-button--active' : '',
    ]"
    @focusout="onFocusOut"
  >
    <button
      ref="triggerEl"
      type="button"
      class="float-button__trigger"
      data-ui="button"
      :title="title"
      aria-haspopup="menu"
      :aria-expanded="modelValue"
      @click.capture.stop="triggerClicked"
      @keydown.down.prevent="openFromKeyboard"
      @keydown.esc.prevent="closeFromKeyboard"
      @mouseenter="mouseEnter"
      @mouseleave="mouseLeave"
    >
      <slot />
    </button>
    <div
      class="float-button__buttons"
      :role="modelValue ? 'menu' : undefined"
      @keydown="onMenuKeydown"
      @mouseenter="mouseEnter"
      @mouseleave="mouseLeave"
    >
      <Transition v-for="(b, i) in buttons" :key="i" name="fade-scale">
        <button
          ref="buttonEls"
          type="button"
          v-show="modelValue"
          class="float-button__button"
          data-ui="button"
          role="menuitem"
          :title="s(b.title)"
          @click="buttonClicked(b, i)"
          @keydown.esc.prevent="closeFromKeyboard"
        >
          <slot v-if="$slots[b.slot]" :name="b.slot"></slot>
          <template v-else>
            <Icon v-if="b.icon" :name="b.icon" />
          </template>
        </button>
      </Transition>
    </div>
  </div>
</template>
<script setup lang="ts">
import type { FloatButtonItem, FloatButtonClickEventData } from '.'
import { nextTick, ref } from 'vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: true,
  },
  title: {
    type: String,
  },
  buttons: {
    type: Array as PropType<FloatButtonItem[]>,
    default: () => [],
  },
  position: {
    type: String,
    default: 'top',
  },
  autoClose: {
    type: Boolean,
  },
})

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'click', d: FloatButtonClickEventData): void
}>()

let leaveTimer: number | undefined
let enterTimer: number | undefined
const triggerEl = ref<HTMLButtonElement>()
const buttonEls = ref<HTMLButtonElement[]>([])

const mouseEnter = () => {
  clearTimeout(leaveTimer)
  clearTimeout(enterTimer)
  enterTimer = window.setTimeout(() => {
    const show = true
    emit('update:modelValue', show)
  }, 0)
}

const mouseLeave = () => {
  clearTimeout(leaveTimer)
  clearTimeout(enterTimer)
  leaveTimer = setTimeout(() => {
    const show = false
    emit('update:modelValue', show)
  }, 100) as unknown as number
}

const triggerClicked = (e: MouseEvent) => {
  clearTimeout(leaveTimer)
  clearTimeout(enterTimer)
  const show = !props.modelValue
  emit('update:modelValue', show)
  if (show && e.detail === 0) {
    nextTick(() => buttonEls.value[0]?.focus())
  }
}

const buttonClicked = (button: FloatButtonItem, index: number) => {
  clearTimeout(enterTimer)
  emit('update:modelValue', false)
  emit('click', { button, index })
}

const openFromKeyboard = () => {
  clearTimeout(enterTimer)
  emit('update:modelValue', true)
  nextTick(() => buttonEls.value[0]?.focus())
}

const closeFromKeyboard = () => {
  clearTimeout(enterTimer)
  clearTimeout(leaveTimer)
  emit('update:modelValue', false)
  nextTick(() => triggerEl.value?.focus())
}

const onMenuKeydown = (e: KeyboardEvent) => {
  const items = buttonEls.value.filter((button) => !button.disabled)
  if (!items.length) return
  const activeIndex = items.indexOf(document.activeElement as HTMLButtonElement)
  let nextIndex: number | undefined
  if (e.key === 'ArrowDown') nextIndex = (activeIndex + 1) % items.length
  else if (e.key === 'ArrowUp') {
    nextIndex = (activeIndex - 1 + items.length) % items.length
  } else if (e.key === 'Home') nextIndex = 0
  else if (e.key === 'End') nextIndex = items.length - 1
  if (nextIndex !== undefined) {
    e.preventDefault()
    items[nextIndex].focus()
  }
}

const onFocusOut = (e: FocusEvent) => {
  const next = e.relatedTarget
  if (next instanceof Node && triggerEl.value?.parentElement?.contains(next)) {
    return
  }
  clearTimeout(enterTimer)
  clearTimeout(leaveTimer)
  emit('update:modelValue', false)
}
</script>
<style lang="scss">
.float-button {
  position: relative;
  width: 60px;
  height: 60px;
}

.float-button__button,
.float-button__trigger {
  display: inline-block;
  width: 100%;
  height: 100%;
  background-color: transparent;
  border: none;
  padding: 0;
  margin: 0;
  outline: none;
  font-size: 50px;
  cursor: pointer;
}

.float-button__buttons {
  position: absolute;
  width: 100%;
}

.float-button__button {
  margin-bottom: 20px;
  transition: transform var(--motion-duration-fast)
    var(--motion-easing-standard);

  &:hover {
    transform: scale(1.2);
  }
}

.float-button__posi-top .float-button__buttons {
  bottom: 0;
  left: 0;
  right: 0;
  margin-bottom: 100%;
}
</style>
