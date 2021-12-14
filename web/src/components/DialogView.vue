<template>
  <div
    class="dialog-view dialog-view__overlay"
    v-if="overlayShowing"
    ref="overlay"
    @click="overlayClicked"
  >
    <transition :name="transition" @after-leave="onDialogClosed">
      <div class="dialog-view__content" v-if="contentShowing">
        <div v-if="$slots.header || title" class="dialog-view__header">
          <slot name="header">
            <span>{{ title }}</span>
          </slot>
          <button
            v-if="closeable"
            class="dialog-view__close-button plain-button"
            @click="closeButtonClicked"
          >
            <i-icon svg="#icon-close" />
          </button>
        </div>
        <div class="dialog-view__body">
          <slot />
        </div>
        <div v-if="$slots.footer" class="dialog-view__footer">
          <slot name="footer" />
        </div>
      </div>
    </transition>
  </div>
</template>
<script>
let scrollLockedCount = 0

export default {
  name: 'DialogView',
  model: {
    prop: 'show',
    event: 'input',
  },
  props: {
    show: {
      type: Boolean,
    },
    title: {
      type: [String, Object],
    },
    transition: {
      type: String,
      default: 'fade',
    },
    overlayClose: {
      type: Boolean,
    },
    escClose: {
      type: Boolean,
    },
    closeable: {
      type: Boolean,
      default: true,
    },
    lockScroll: {
      type: Boolean,
      default: true,
    },
  },
  watch: {
    show: {
      immediate: true,
      handler(val) {
        if (val) {
          this.overlayShowing = val
          this.$nextTick(() => {
            this.contentShowing = true
          })
          this.setupEvents()
        } else {
          this.contentShowing = false
          if (!this.transition) {
            this.overlayShowing = false
          }
          this.removeEvents()
        }
      },
    },
    overlayShowing: {
      immediate: true,
      handler(val) {
        this.onDialogVisibleChanged(val)
      },
    },
  },
  data() {
    return {
      overlayShowing: false,
      contentShowing: false,
    }
  },
  beforeDestroy() {
    this.onDialogVisibleChanged(false)
    this.removeEvents()
  },
  methods: {
    overlayClicked(e) {
      if (!this.overlayClose) return
      if (this.closeable && e.target === this.$refs.overlay) {
        this.close()
      }
    },
    closeButtonClicked() {
      this.close()
    },
    onKeyDown(e) {
      if (!this.escClose) return
      if (this.closeable && e.key === 'Escape' && this.show) {
        this.close()
        e.preventDefault()
      }
    },
    close() {
      this.$emit('input', false)
    },
    onDialogClosed() {
      this.overlayShowing = false
      this.$emit('closed')
    },
    setupEvents() {
      window.addEventListener('keydown', this.onKeyDown)
    },
    removeEvents() {
      window.removeEventListener('keydown', this.onKeyDown)
    },
    onDialogVisibleChanged(showing) {
      if (!this.lockScroll) return
      if (showing) {
        if (this._scrollLocked) return
        scrollLockedCount++
        this._scrollLocked = true
      } else {
        if (!this._scrollLocked) return
        scrollLockedCount--
        this._scrollLocked = false
      }
      if (scrollLockedCount > 0) {
        document.body.classList.add('dialog-view--scrollable-locked')
      } else {
        document.body.classList.remove('dialog-view--scrollable-locked')
      }
    },
  },
}
</script>
<style lang="scss">
.dialog-view--scrollable-locked {
  overflow: hidden;
}

.dialog-view__overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  overflow: hidden;
  @include var(background-color, dialog-overlay-bg-color);
  z-index: 1000;

  display: flex;
  justify-content: center;
  align-items: center;
}

.dialog-view__content {
  @include var(background-color, secondary-bg-color);
  @include var(box-shadow, dialog-content-shadow);
}

.dialog-view__header {
  position: relative;
  min-width: 200px;
  font-size: 28px;
  font-weight: normal;
  user-select: none;
  padding: 16px 48px 16px 16px;
}

.dialog-view__close-button {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  right: 12px;
  white-space: nowrap;
  overflow: hidden;
}

.dialog-view__body {
  overflow: hidden;
}
</style>
