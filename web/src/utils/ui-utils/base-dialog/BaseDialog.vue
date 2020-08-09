<template>
  <dialog-view
    class="base-dialog"
    :title="title"
    :show="showing"
    :transition="transition"
    @input="$emit('close')"
    @closed="$emit('closed')"
    esc-close
    overlay-close
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
        <button
          v-if="cancelText"
          class="simple-button base-dialog__button-cancel"
          :class="{ loading, [cancelType]: true }"
          @click="$emit('cancel')"
          :disabled="!!loading"
        >
          <i-icon v-if="loading === 'cancel'" svg="#icon-loading" />
          <template v-else>{{ cancelText }}</template>
        </button>
        <button
          class="simple-button base-dialog__button-ok"
          :class="{ loading, [confirmType]: true }"
          @click="$emit('confirm')"
          :disabled="!!loading"
        >
          <i-icon v-if="loading === 'confirm'" svg="#icon-loading" />
          <template v-else>{{ confirmText }}</template>
        </button>
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
      type: String,
      required: true
    },
    confirmText: {
      type: String,
      required: true
    },
    confirmType: {
      type: String
    },
    cancelText: {
      type: String
    },
    cancelType: {
      type: String,
      default: 'info'
    },
    transition: {
      type: String
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
