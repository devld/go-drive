<template>
  <DialogView
    v-model:show="showing"
    class="loading-dialog"
    transition="none"
    :accessible-label="text || t('app.loading')"
  >
    <div class="loading-dialog__content">
      <LoadingState variant="dialog" :text="text">
        <SimpleButton
          v-if="cancelText"
          class="loading-dialog__cancel"
          :type="cancelType"
          :loading="cancelLoading"
          @click="cancel"
          >{{ cancelText }}</SimpleButton
        >
      </LoadingState>
    </div>
  </DialogView>
</template>
<script setup lang="ts">
import { SimpleButtonType } from '@/components/SimpleButton'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LoadingOptions } from '.'
import LoadingState from '@/components/LoadingState.vue'

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
  background-color: var(--color-loading-overlay);
  z-index: 9999;

  .dialog-view__content {
    box-shadow: none;
    background-color: transparent;
  }
}

.loading-dialog__content {
  display: flex;
  justify-content: center;
  align-items: center;
}

.loading-dialog__cancel {
  margin-top: 4px;
}
</style>
