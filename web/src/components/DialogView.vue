<template>
  <div class="dialog-view dialog-view__overlay" v-if="overlayShowing">
    <transition v-if="transition" :name="transition" @after-leave="overlayShowing = false">
      <div ref="overlay" class="dialog-view__content" v-if="contentShowing" @click="overlayClicked">
        <slot />
      </div>
    </transition>
    <div class="dialog-view__content" v-else-if="contentShowing">
      <slot />
    </div>
  </div>
</template>
<script>
export default {
  name: 'DialogView',
  model: {
    prop: 'show',
    event: 'input'
  },
  props: {
    show: {
      type: Boolean
    },
    transition: {
      type: String,
      default: 'top-fade'
    },
    overlayClose: {
      type: Boolean
    }
  },
  watch: {
    show: {
      immediate: true,
      handler (val) {
        if (val) {
          this.overlayShowing = val
          this.$nextTick(() => {
            this.contentShowing = true
          })
        } else {
          this.contentShowing = false
          if (!this.transition) {
            this.overlayShowing = false
          }
        }
      }
    }
  },
  data () {
    return {
      overlayShowing: false,
      contentShowing: false
    }
  },
  methods: {
    overlayClicked (e) {
      if (!this.overlayClose) return
      if (e.target === this.$refs.overlay) {
        this.$emit('input', false)
      }
    }
  }
}
</script>
<style lang="scss">
.dialog-view__overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  overflow: hidden;
  background-color: rgba(0, 0, 0, 0.1);
}

.dialog-view__content {
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
}
</style>
