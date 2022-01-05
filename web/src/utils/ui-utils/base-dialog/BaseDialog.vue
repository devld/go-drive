<template>
  <dialog-view
    v-focus
    class="base-dialog"
    :title="title"
    :show="showing"
    :transition="transition"
    :esc-close="escClose"
    :overlay-close="overlayClose"
    :closeable="!loading"
    tabindex="-1"
    @update:show="emit('close')"
    @closed="emit('closed')"
    @keydown.enter="emit('confirm')"
  >
    <div class="base-dialog__content-wrapper">
      <slot />
    </div>

    <template #footer>
      <div class="base-dialog__footer">
        <simple-button
          v-if="cancelText"
          class="base-dialog__button-cancel"
          :loading="loading === 'cancel'"
          :type="cancelType"
          :disabled="!!loading"
          @click="emit('cancel')"
          >{{ cancelText }}</simple-button
        >
        <simple-button
          ref="confirmButton"
          class="base-dialog__button-ok"
          :loading="loading === 'confirm'"
          :type="confirmType"
          :disabled="!!loading || confirmDisabled"
          @click="emit('confirm')"
          >{{ confirmText }}</simple-button
        >
      </div>
    </template>
  </dialog-view>
</template>
<script setup>
defineProps({
  showing: {
    type: Boolean,
    required: true,
  },
  loading: {
    type: String,
    required: true,
  },
  title: {
    type: [String, Object],
    required: true,
  },
  confirmText: {
    type: [String, Object],
    required: true,
  },
  confirmType: {
    type: String,
  },
  confirmDisabled: {
    type: Boolean,
  },
  cancelText: {
    type: [String, Object],
  },
  cancelType: {
    type: String,
    default: 'info',
  },
  transition: {
    type: String,
  },
  escClose: {
    type: Boolean,
  },
  overlayClose: {
    type: Boolean,
  },
})
const emit = defineEmits(['close', 'closed', 'confirm', 'cancel'])
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
