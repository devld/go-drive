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
  font-size: 16px;
  border: none;
  background-color: #409eff;
  color: #fff;
  padding: 4px 10px;
  cursor: pointer;
  transition: 0.3s;
  user-select: none;

  &:hover {
    box-shadow: 0 0 4px rgba(0, 0, 0, 0.2);
  }

  &.small {
    font-size: 14px;
    padding: 4px 6px;
  }

  &[disabled] {
    cursor: not-allowed;
    background-color: #a0cfff;
  }

  &.info {
    background-color: #909399;

    &[disabled] {
      background-color: #c8c9cc;
    }
  }

  &.success {
    background-color: #67c23a;

    &[disabled] {
      background-color: #b3e19d;
    }
  }

  &.warning {
    background-color: #e6a23c;

    &[disabled] {
      background-color: #f3d19e;
    }
  }

  &.danger {
    background-color: #f56c6c;

    &[disabled] {
      background-color: #fab6b6;
    }
  }

  &.loading .icon {
    animation: spinning 1s linear infinite;
  }
}
</style>
