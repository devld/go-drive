<template>
  <dialog-view class="loading-dialog" v-model="showing" transition="none">
    <div class="loading-dialog__content">
      <i-icon class="loading-dialog__icon" svg="#icon-loading" />
      <span class="loading-dialog__text">{{ text }}</span>
      <simple-button
        class="loading-dialog__cancel"
        v-if="cancelText"
        :type="cancelType"
        :loading="cancelLoading"
        @click="cancel"
        >{{ cancelText }}</simple-button
      >
    </div>
  </dialog-view>
</template>
<script>
export default {
  name: 'LoadingDialog',
  data() {
    return {
      showing: false,
      text: '',
      cancelText: '',
      cancelType: '',

      cancelLoading: false,
    }
  },
  methods: {
    show(opts = {}) {
      this.text = opts.text || ''

      this._cancelCallback = opts.onCancel

      this.cancelText = this._cancelCallback
        ? opts.cancelText || this.$t('dialog.loading.cancel')
        : ''
      this.cancelType = opts.cancelType || 'info'

      this.showing = true
    },
    hide() {
      this.showing = false
    },
    async cancel() {
      this.cancelLoading = true
      try {
        await this._cancelCallback()
        this.hide()
      } catch (e) {
        /* nothing */
      } finally {
        this.cancelLoading = false
      }
    },
  },
}
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
