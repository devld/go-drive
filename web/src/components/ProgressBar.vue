<template>
  <div class="progress-bar">
    <div
      v-if="typeof progress === 'number'"
      class="progress-bar__inner"
      :style="{ width: `${progress || 0}%` }"
    ></div>
  </div>
</template>
<script setup>
import { ref, watchEffect } from 'vue'

const FAKE_START = 10
const FAKE_FREEZE = 90

const props = defineProps({
  show: {
    type: [Number, Boolean],
    required: true,
  },
})

const progress = ref(0)

let timer
let timer1

const clearTimer = () => {
  clearTimeout(timer1)
  clearInterval(timer)
}

const setTimer = () => {
  clearTimer()
  timer = setInterval(() => {
    if (progress.value <= FAKE_FREEZE) {
      progress.value += 1
    }
  }, 100)
}

const startFakeProgress = () => {
  progress.value = FAKE_START
  setTimer()
}

const completeFakeProgress = () => {
  clearTimer()
  progress.value = 100
  timer1 = setTimeout(() => {
    progress.value = null
  }, 400)
}

watchEffect(() => {
  const v = props.show
  if (typeof v === 'number') {
    clearTimer()
    progress.value = v
  } else if (typeof v === 'boolean') {
    if (v) {
      startFakeProgress()
    } else {
      completeFakeProgress()
    }
  }
})
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
