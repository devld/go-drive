<template>
  <button
    class="simple-button"
    :class="{ loading, [type!]: !!type, small }"
    :disabled="!!loading || disabled"
    :type="nativeType"
    @click="emit('click', $event)"
  >
    <Icon v-if="loading" svg="#icon-loading" />
    <template v-else>
      <Icon v-if="icon" :svg="icon" />
      <slot />
    </template>
  </button>
</template>
<script setup lang="ts">
import type { SimpleButtonType, SimpleButtonNativeType } from '.'

defineProps({
  loading: {
    type: Boolean,
  },
  type: {
    type: String as PropType<SimpleButtonType>,
  },
  small: {
    type: Boolean,
  },
  icon: {
    type: String,
  },
  disabled: {
    type: Boolean,
  },
  nativeType: {
    type: String as PropType<SimpleButtonNativeType>,
  },
})

const emit = defineEmits<{ (e: 'click', event: MouseEvent): void }>()
</script>
<style lang="scss">
.simple-button {
  font-size: 16px;
  border: none;
  background-color: var(--btn-bg-color-default);
  color: var(--btn-color-default);
  padding: 4px 10px;
  cursor: pointer;
  transition: 0.3s;
  user-select: none;
  -webkit-user-select: none;
  line-height: 20px;

  & + .simple-button {
    margin-left: 0.5em;
  }

  &:hover {
    box-shadow: 0 0 4px rgba(0, 0, 0, 0.2);
  }

  &.small {
    font-size: 14px;
    padding: 4px 6px;
  }

  &[disabled] {
    cursor: not-allowed;
    background-color: var(--btn-bg-color-disabled-default);
  }

  $types: ('info', 'success', 'warning', 'danger');
  @each $type in $types {
    &.#{$type} {
      background-color: var(--btn-bg-color-#{$type});
      color: var(--btn-color-#{$type});

      &[disabled] {
        background-color: var(--btn-bg-color-disabled-#{$type});
        color: var(--btn-color-disabled-#{$type});
      }
    }
  }

  &.loading .icon {
    animation: spinning 1s linear infinite;
  }
}
</style>
