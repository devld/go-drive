<template>
  <button
    class="simple-button"
    :class="{ loading, [type]: !!type, small }"
    @click="$emit('click')"
    :disabled="!!loading || disabled"
    :type="nativeType"
  >
    <i-icon v-if="loading" svg="#icon-loading" />
    <template v-else>
      <i-icon v-if="icon" :svg="icon" />
      <slot />
    </template>
  </button>
</template>
<script>
export default {
  name: 'SimpleButton',
  props: {
    loading: {
      type: Boolean,
    },
    type: {
      type: String,
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
      type: String,
    },
  },
}
</script>
<style lang="scss">
.simple-button {
  font-size: 16px;
  border: none;
  @include var(background-color, btn-bg-color-default);
  @include var(color, btn-color-default);
  padding: 4px 10px;
  cursor: pointer;
  transition: 0.3s;
  user-select: none;
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
    @include var(background-color, btn-bg-color-disabled-default);
  }

  $types: ('info', 'success', 'warning', 'danger');
  @each $type in $types {
    &.#{$type} {
      @include var(background-color, btn-bg-color-#{$type});
      @include var(color, btn-color-#{$type});

      &[disabled] {
        @include var(background-color, btn-bg-color-disabled-#{$type});
        @include var(color, btn-color-disabled-#{$type});
      }
    }
  }

  &.loading .icon {
    animation: spinning 1s linear infinite;
  }
}
</style>
