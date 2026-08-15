<template>
  <button
    class="simple-button"
    :class="{ loading, [type!]: !!type, [variant!]: !!variant, small }"
    data-ui="button"
    :data-variant="variant === 'plain' ? 'plain' : type || 'primary'"
    :data-size="small ? 'compact' : 'default'"
    :disabled="!!loading || disabled"
    :type="nativeType"
    :aria-busy="loading ? 'true' : undefined"
    @click="emit('click', $event)"
  >
    <LoadingIndicator
      v-if="loading"
      class="simple-button__loading-indicator"
    />
    <span class="simple-button__content" :class="{ hidden: loading }">
      <Icon v-if="icon" :name="icon" aria-hidden="true" />
      <slot />
    </span>
  </button>
</template>
<script setup lang="ts">
import type {
  SimpleButtonType,
  SimpleButtonVariant,
  SimpleButtonNativeType,
} from '.'
import type { IconName } from '@/components/icons'
import LoadingIndicator from '@/components/LoadingIndicator.vue'

defineProps({
  loading: {
    type: Boolean,
  },
  type: {
    type: String as PropType<SimpleButtonType>,
  },
  variant: {
    type: String as PropType<SimpleButtonVariant>,
    default: 'solid',
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
  position: relative;
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

  &.loading[disabled] {
    opacity: 0.86;
  }

  &.outline {
    border: 1px solid var(--color-accent);
    background-color: transparent;
    color: var(--color-accent);
  }

  &.plain {
    margin: 0;
    border: 0;
    border-radius: 0;
    padding: 0;
    background-color: transparent;
    color: var(--color-text);
    font: inherit;
    font-size: 28px;
    text-decoration: none;
  }

  &.plain.small {
    font-size: 16px;
  }

  &.plain:hover {
    box-shadow: none;
  }

  $types: ('info', 'success', 'warning', 'danger');
  @each $type in $types {
    &.#{$type} {
      background-color: var(--color-#{$type}-strong);
      color: var(--color-on-#{$type});
    }

    &.#{$type}.outline {
      border-color: var(--color-#{$type}-strong);
      background-color: transparent;
      color: var(--color-#{$type}-strong);
    }

    &.#{$type}.plain {
      background-color: transparent;
      color: var(--color-#{$type}-strong);
    }
  }
}

.simple-button__content.hidden {
  visibility: hidden;
}

.simple-button__loading-indicator {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 1em;
  height: 1em;
  margin: -0.5em 0 0 -0.5em;
}
</style>
