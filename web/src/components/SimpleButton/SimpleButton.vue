<template>
  <button
    class="simple-button"
    :class="{ loading, [type!]: !!type, small }"
    data-ui="button"
    :data-variant="type || 'primary'"
    :data-size="small ? 'compact' : 'default'"
    :disabled="!!loading || disabled"
    :type="nativeType"
    :aria-busy="loading ? 'true' : undefined"
    @click="emit('click', $event)"
  >
    <Icon v-if="loading" class="loading-icon" name="loading" aria-hidden="true" />
    <template v-else>
      <Icon v-if="icon" :name="icon" aria-hidden="true" />
      <slot />
    </template>
  </button>
</template>
<script setup lang="ts">
import type { SimpleButtonType, SimpleButtonNativeType } from '.'
import type { IconName } from '@/components/icons'

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
    type: String as PropType<IconName>,
  },
  disabled: {
    type: Boolean,
  },
  nativeType: {
    type: String as PropType<SimpleButtonNativeType>,
    default: 'button',
  },
})

const emit = defineEmits<{ (e: 'click', event: MouseEvent): void }>()
</script>
<style lang="scss">
.simple-button {
  font-size: 16px;
  border: none;
  background-color: var(--color-accent-strong);
  color: var(--color-on-accent);
  padding: 4px 10px;
  cursor: pointer;
  transition:
    box-shadow var(--motion-duration-fast) var(--motion-easing-standard),
    opacity var(--motion-duration-fast) var(--motion-easing-standard);
  -webkit-user-select: none;
  user-select: none;
  line-height: 20px;
  border-radius: var(--radius-control);

  & + .simple-button {
    margin-left: 0.5em;
  }

  &:hover {
    box-shadow: var(--shadow-control);
  }

  &.small {
    font-size: 14px;
    padding: 4px 6px;
  }

  &[disabled] {
    cursor: not-allowed;
    opacity: 0.58;
  }

  $types: ('info', 'success', 'warning', 'danger');
  @each $type in $types {
    &.#{$type} {
      background-color: var(--color-#{$type}-strong);
      color: var(--color-on-#{$type});
    }
  }
}
</style>
