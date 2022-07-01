<template>
  <div class="simple-dropdown" :class="{ active }" tabindex="-1" @blur="onBlur">
    <span
      class="simple-dropdown__trigger"
      role="button"
      @click="triggerClicked"
    >
      <slot />
    </span>
    <Transition :name="transition">
      <div v-show="active" class="simple-dropdown__dropdown">
        <slot name="dropdown" />
      </div>
    </Transition>
  </div>
</template>
<script setup lang="ts">
import { ref, watchEffect } from 'vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
  },
  transition: {
    type: String,
    default: 'top-fade',
  },
})
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const active = ref(false)

watchEffect(() => {
  active.value = !!props.modelValue
})

const emitInput = () => {
  emit('update:modelValue', active.value)
}

const triggerClicked = () => {
  active.value = !active.value
  emitInput()
}

const onBlur = () => {
  active.value = false
  emitInput()
}
</script>
<style lang="scss">
.simple-dropdown {
  display: inline-block;
  position: relative;
}

.simple-dropdown__trigger {
  cursor: pointer;
}

.simple-dropdown__dropdown {
  position: absolute;
  right: 0;
  margin-top: 10px;
  z-index: 999;
  background-color: var(--secondary-bg-color);
  box-shadow: var(--dialog-content-shadow);
}
</style>
