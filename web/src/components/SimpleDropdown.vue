<template>
  <div class="simple-dropdown" :class="{ active }" tabindex="-1" @blur="onBlur">
    <span
      class="simple-dropdown__trigger"
      @click="triggerClicked"
      role="button"
    >
      <slot />
    </span>
    <transition name="top-fade">
      <div class="simple-dropdown__dropdown" v-show="active">
        <slot name="dropdown" />
      </div>
    </transition>
  </div>
</template>
<script>
export default {
  name: 'SimpleDropdown',
  props: {
    value: {
      type: Boolean,
    },
  },
  watch: {
    value: {
      immediate: true,
      handler(val) {
        this.active = !!val
      },
    },
    transition: {
      type: String,
      default: 'top-fade',
    },
  },
  mounted() {},
  beforeDestroy() {},
  data() {
    return {
      active: false,
    }
  },
  methods: {
    triggerClicked() {
      this.active = !this.active
      this.emitInput()
    },
    onBlur() {
      this.active = false
      this.emitInput()
    },
    emitInput() {
      this.$emit('input', this.active)
    },
  },
}
</script>
<style lang="scss">
.simple-dropdown {
  display: inline-block;
  position: relative;
}

.simple-dropdown__trigger {
  cursor: pointer;
}

.simple-dropdown__dropdown {
  position: absolute;
  right: 0;
  margin-top: 10px;
  z-index: 999;
  @include var(background-color, secondary-bg-color);
  @include var(box-shadow, dialog-content-shadow);
}
</style>
