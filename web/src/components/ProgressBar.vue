<template>
  <div class="progress-bar">
    <div
      v-if="typeof progress === 'number'"
      class="progress-bar__inner"
      :style="{ width: `${progress || 0}%` }"
    ></div>
  </div>
</template>
<script>
const FAKE_START = 10
const FAKE_FREEZE = 90

export default {
  name: 'ProgressBar',
  props: {
    value: {
      type: [Number, Boolean],
      required: true,
    },
  },
  data() {
    return {
      progress: 0,
    }
  },
  watch: {
    value: {
      immediate: true,
      handler(v) {
        if (typeof v === 'number') {
          this.clearTimer()
          this.progress = v
        }
        if (typeof v === 'boolean') {
          if (v) {
            this.startFakeProgress()
          } else {
            this.completeFakeProgress()
          }
        }
      },
    },
  },
  methods: {
    startFakeProgress() {
      this.progress = FAKE_START
      this.setTimer()
    },
    completeFakeProgress() {
      this.clearTimer()
      this.progress = 100
      this._tt = setTimeout(() => {
        this.progress = null
      }, 400)
    },

    setTimer() {
      this.clearTimer()
      this._t = setInterval(() => {
        if (this.progress <= FAKE_FREEZE) {
          this.progress += 1
        }
      }, 100)
    },
    clearTimer() {
      clearTimeout(this._tt)
      clearInterval(this._t)
    },
  },
}
</script>
<style lang="scss">
.progress-bar {
  height: 2px;
}

.progress-bar__inner {
  height: 100%;
  background-color: #66ccff;
  transition: all 0.4s;
}
</style>
