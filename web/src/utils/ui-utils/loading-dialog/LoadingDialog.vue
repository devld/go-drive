<template>
  <DialogView v-model:show="showing" class="loading-dialog" transition="none">
    <div class="loading-dialog__content">
      <Icon class="loading-dialog__icon loading-icon" svg="#icon-loading" />
      <span class="loading-dialog__text">{{ text }}</span>
      <SimpleButton
        v-if="cancelText"
        class="loading-dialog__cancel"
        :type="cancelType"
        :loading="cancelLoading"
        @click="cancel"
        >{{ cancelText }}</SimpleButton
      >
    </div>
  </DialogView>
</template>
<script setup lang="ts">
import { SimpleButtonType } from '@/components/SimpleButton'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LoadingOptions } from '.'

const showing = ref(false)
const text = ref<I18nText>('')
const cancelText = ref<I18nText>('')
const cancelType = ref<SimpleButtonType | undefined>(undefined)
const cancelLoading = ref(false)

const { t } = useI18n()

let cancelCallback: (() => PromiseValue<void>) | undefined

const show = (opts: LoadingOptions = {}) => {
  text.value = opts.text ?? ''

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
    cancelCallback && (await cancelCallback())
    hide()
  } catch (e: any) {
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
  -webkit-user-select: none;
  user-select: none;
  margin-top: 1em;
  word-break: break-all;
}

.loading-dialog__cancel {
  margin-top: 1em;
}

.icon.loading-dialog__icon {
  width: 5vw;
  height: 5vw;
  color: var(--secondary-text-color);
}
</style>
