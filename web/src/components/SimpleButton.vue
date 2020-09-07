<template>
  <button
    class="simple-button"
    :class="{ loading, [type]: true, small }"
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
      type: Boolean
    },
    type: {
      type: String
    },
    small: {
      type: Boolean
    },
    icon: {
      type: String
    },
    disabled: {
      type: Boolean
    },
    nativeType: {
      type: String
    }
  }
}
</script>
<style lang="scss">
.simple-button {
  &.loading .icon {
    animation: spinning 1s linear infinite;
  }
}
</style>
