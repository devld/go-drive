<template>
  <dialog-view
    class="base-dialog"
    :title="title"
    :show="showing"
    :transition="transition"
    @input="$emit('close')"
    @closed="$emit('closed')"
    :esc-close="escClose"
    :overlay-close="overlayClose"
    :closeable="!loading"
    @keydown.13.native="$emit('confirm')"
    tabindex="-1"
    v-focus
  >
    <div class="base-dialog__content-wrapper">
      <slot />
    </div>

    <template slot="footer">
      <div class="base-dialog__footer">
        <simple-button
          v-if="cancelText"
          class="base-dialog__button-cancel"
          :loading="loading === 'cancel'"
          :type="cancelType"
          @click="$emit('cancel')"
          :disabled="!!loading"
          >{{ cancelText }}</simple-button
        >
        <simple-button
          ref="confirmButton"
          class="base-dialog__button-ok"
          :loading="loading === 'confirm'"
          :type="confirmType"
          @click="$emit('confirm')"
          :disabled="!!loading || confirmDisabled"
          >{{ confirmText }}</simple-button
        >
      </div>
    </template>
  </dialog-view>
</template>
<script>

export default {
  name: 'BaseDialog',
  props: {
    showing: {
      type: Boolean,
      required: true
    },
    loading: {
      type: String,
      required: true
    },
    title: {
      type: [String, Object],
      required: true
    },
    confirmText: {
      type: [String, Object],
      required: true
    },
    confirmType: {
      type: String
    },
    confirmDisabled: {
      type: Boolean
    },
    cancelText: {
      type: [String, Object]
    },
    cancelType: {
      type: String,
      default: 'info'
    },
    transition: {
      type: String
    },
    escClose: {
      type: Boolean
    },
    overlayClose: {
      type: Boolean
    }
  }
}
</script>
<style lang="scss">
.base-dialog__content-wrapper {
  text-align: center;
  padding: 16px;
}

.base-dialog__footer {
  padding: 16px;
  text-align: right;

  button:not(:last-child) {
    margin-right: 10px;
  }
}

.base-dialog__button-ok {
  &.loading .icon {
    animation: spinning 1s linear infinite;
  }
}
</style>
