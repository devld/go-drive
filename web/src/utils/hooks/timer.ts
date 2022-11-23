import { onUnmounted, onMounted, Ref, watch } from 'vue'

export const useInterval = (
  callback: Fn<void>,
  interval: number | Ref<number>,
  immediate: boolean
) => {
  let t: number

  const getInterval = () =>
    typeof interval === 'number' ? interval : interval.value

  const startInterval = () => {
    clearInterval(t)
    const interval = getInterval()
    if (interval > 0) {
      t = setInterval(() => {
        callback()
      }, interval) as unknown as number
    }
  }

  onMounted(() => {
    if (immediate) {
      callback()
    }
    startInterval()
  })

  if (typeof interval !== 'number') {
    watch(interval, startInterval)
  }

  onUnmounted(() => {
    clearInterval(t)
  })
}
