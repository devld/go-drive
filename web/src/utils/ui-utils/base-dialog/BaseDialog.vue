<template>
  <DialogView
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
        <SimpleButton
          v-if="cancelText"
          class="base-dialog__button-cancel"
          :loading="loading === 'cancel'"
          :type="cancelType"
          :disabled="!!loading"
          @click="emit('cancel')"
          >{{ cancelText }}</SimpleButton
        >
        <SimpleButton
          ref="confirmButton"
          class="base-dialog__button-ok"
          :loading="loading === 'confirm'"
          :type="confirmType"
          :disabled="!!loading || confirmDisabled"
          @click="emit('confirm')"
          >{{ confirmText }}</SimpleButton
        >
      </div>
    </template>
  </DialogView>
</template>
<script setup lang="ts">
import { SimpleButtonType } from '@/components/SimpleButton'

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
    type: [String, Object] as PropType<I18nText>,
    required: true,
  },
  confirmText: {
    type: [String, Object] as PropType<I18nText>,
    required: true,
  },
  confirmType: {
    type: String as PropType<SimpleButtonType>,
  },
  confirmDisabled: {
    type: Boolean,
  },
  cancelText: {
    type: [String, Object] as PropType<I18nText>,
  },
  cancelType: {
    type: String as PropType<SimpleButtonType>,
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
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'closed'): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()
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
