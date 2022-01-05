<template>
  <div
    class="float-button"
    :class="[
      `float-button__posi-${position}`,
      modelValue ? 'float-button--active' : '',
    ]"
  >
    <div
      class="float-button__buttons"
      @mouseenter="mouseEnter"
      @mouseleave="mouseLeave"
    >
      <transition v-for="(b, i) in buttons" :key="i" name="scale-fade">
        <button
          v-show="modelValue"
          class="float-button__button"
          :title="b.title"
          @click="buttonClicked(b, i)"
        >
          <slot v-if="$slots[b.slot]" :name="b.slot"></slot>
          <template v-else>
            <i class="iconfont" v-html="b.icon"></i>
          </template>
        </button>
      </transition>
    </div>
    <button
      class="float-button__trigger"
      :title="title"
      @click.capture.stop="triggerClicked"
      @mouseenter="mouseEnter"
      @mouseleave="mouseLeave"
    >
      <slot />
    </button>
  </div>
</template>
<script setup>
const props = defineProps({
  modelValue: {
    type: Boolean,
    default: true,
  },
  title: {
    type: String,
  },
  buttons: {
    type: Array,
    default() {
      return []
    },
  },
  position: {
    type: String,
    default: 'top',
  },
  autoClose: {
    type: Boolean,
  },
})

const emit = defineEmits(['update:modelValue', 'click'])

let leaveTimer

const mouseEnter = () => {
  clearTimeout(leaveTimer)
  setTimeout(() => {
    const show = true
    emit('update:modelValue', show)
  }, 0)
}

const mouseLeave = () => {
  clearTimeout(leaveTimer)
  leaveTimer = setTimeout(() => {
    const show = false
    emit('update:modelValue', show)
  }, 100)
}

const triggerClicked = () => {
  clearTimeout(leaveTimer)
  const show = !props.modelValue
  emit('update:modelValue', show)
}

const buttonClicked = (button, index) => {
  emit('update:modelValue', false)
  emit('click', { button, index })
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

  &:hover {
    .float-button__button {
      transition: 0.4s;
    }
  }
}

.float-button__button {
  margin-bottom: 20px;

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
