<template>
  <button
    class="simple-button"
    :class="{
      loading,
      [type!]: !!type,
      [variant!]: !!variant,
      small,
      'icon-only': !!icon && !$slots.default,
    }"
    data-ui="button"
    :data-variant="variant === 'plain' ? 'plain' : type || 'primary'"
    :data-size="small ? 'compact' : 'default'"
    :disabled="!!loading || disabled"
    :type="nativeType"
    :aria-busy="loading ? 'true' : undefined"
    @click="emit('click', $event)"
  >
    <span v-if="loading" class="simple-button__loading">
      <LoadingIndicator class="simple-button__loading-indicator" />
      <span v-if="$slots.loading" class="simple-button__loading-text">
        <slot name="loading" />
      </span>
    </span>
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
  --simple-button-pad-y: 4px;
  --simple-button-pad-x: 10px;
  --simple-button-line-height: 20px;
  --simple-button-size: calc(
    var(--simple-button-line-height) + 2 * var(--simple-button-pad-y) + 2px
  );

  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  font-size: 16px;
  border: 1px solid transparent;
  background-color: var(--color-accent-strong);
  color: var(--color-on-accent);
  padding: var(--simple-button-pad-y) var(--simple-button-pad-x);
  cursor: pointer;
  transition:
    box-shadow var(--motion-duration-fast) var(--motion-easing-standard),
    opacity var(--motion-duration-fast) var(--motion-easing-standard);
  -webkit-user-select: none;
  user-select: none;
  line-height: var(--simple-button-line-height);
  border-radius: var(--radius-control);

  & + .simple-button {
    margin-left: 0.5em;
  }

  &:hover {
    box-shadow: var(--shadow-control);
  }

  &.small {
    --simple-button-pad-x: 6px;
    --simple-button-pad-y: 3px;
    --simple-button-line-height: 16px;
    font-size: 14px;
  }

  &:not(.plain).icon-only {
    padding: var(--simple-button-pad-y);
    line-height: 0;
    width: var(--simple-button-size);
    height: var(--simple-button-size);
  }

  &[disabled] {
    cursor: not-allowed;
    opacity: 0.58;
  }

  &.loading[disabled] {
    opacity: 0.86;
  }

  &.outline {
    border-color: var(--color-accent);
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
    line-height: 0;
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

.simple-button__content {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.35em;
  min-width: 0;
  line-height: inherit;

  .icon,
  svg {
    display: block;
    flex-shrink: 0;
  }
}

.simple-button.plain .simple-button__content {
  line-height: normal;

  &:has(> .icon:only-child),
  &:has(> svg:only-child) {
    line-height: 0;
  }
}

.simple-button__content.hidden {
  visibility: hidden;
}

.simple-button__loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.4em;
  padding: inherit;
}

.simple-button__loading-indicator {
  width: 1em;
  height: 1em;
  flex-shrink: 0;
}

.simple-button__loading-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
