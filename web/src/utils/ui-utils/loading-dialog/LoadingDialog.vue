<template>
  <dialog-view v-model:show="showing" class="loading-dialog" transition="none">
    <div class="loading-dialog__content">
      <i-icon class="loading-dialog__icon" svg="#icon-loading" />
      <span class="loading-dialog__text">{{ text }}</span>
      <simple-button
        v-if="cancelText"
        class="loading-dialog__cancel"
        :type="cancelType"
        :loading="cancelLoading"
        @click="cancel"
        >{{ cancelText }}</simple-button
      >
    </div>
  </dialog-view>
</template>
<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const showing = ref(false)
const text = ref('')
const cancelText = ref('')
const cancelType = ref('')
const cancelLoading = ref(false)

const { t } = useI18n()

let cancelCallback

const show = (opts = {}) => {
  text.value = opts.text || ''

  cancelCallback = opts.onCancel

  cancelText.value = cancelCallback
    ? opts.cancelText || t('dialog.loading.cancel')
    : ''
  cancelType.value = opts.cancelType || 'info'

  showing.value = true
}

const hide = () => {
  showing.value = false
}

const cancel = async () => {
  cancelLoading.value = true
  try {
    await cancelCallback()
    hide()
  } catch (e) {
    /* nothing */
  } finally {
    cancelLoading.value = false
  }
}

defineExpose({ show, hide })
</script>
<style lang="scss">
.dialog-view.loading-dialog {
  background-color: var(--loading-overlay-bg-color);
  z-index: 9999;

  .dialog-view__content {
    box-shadow: none;
    background-color: transparent;
  }
}

.loading-dialog__content {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.loading-dialog__text {
  max-width: 50vw;
  user-select: none;
  margin-top: 1em;
  word-break: break-all;
}

.loading-dialog__cancel {
  margin-top: 1em;
}

.icon.loading-dialog__icon {
  width: 10vw;
  height: 10vw;
  animation: spinning 1s linear infinite;
}
</style>
