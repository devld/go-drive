<template>
  <dialog-view class="loading-dialog" v-model="showing" transition="none">
    <div class="loading-dialog__content">
      <i-icon class="loading-dialog__icon" svg="#icon-loading" />
      <span class="loading-dialog__text">{{ text }}</span>
    </div>
  </dialog-view>
</template>
<script>
export default {
  name: 'LoadingDialog',
  data () {
    return {
      showing: false,
      text: ''
    }
  },
  methods: {
    show (opts = {}) {
      this.text = opts.text || ''

      this.showing = true
    },
    hide () {
      this.showing = false
    }
  }
}
</script>
<style lang="scss">
.dialog-view.loading-dialog {
  background-color: rgba(255, 255, 255, 0.4);

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
  margin-top: 1em;
}

.icon.loading-dialog__icon {
  width: 10vw;
  height: 10vw;
  animation: spinning 1s linear infinite;
}
</style>
