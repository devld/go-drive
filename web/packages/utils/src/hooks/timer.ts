import { onMounted, onUnmounted, watch, type Ref } from 'vue'

export function useInterval(
  callback: () => void,
  interval: number | Ref<number>,
  immediate = false
) {
  let timer: number | undefined

  const getInterval = () =>
    typeof interval === 'number' ? interval : interval.value

  const startInterval = () => {
    clearInterval(timer)
    const delay = getInterval()
    if (delay > 0) {
      timer = setInterval(callback, delay) as unknown as number
    }
  }

  onMounted(() => {
    if (immediate) callback()
    startInterval()
  })

  if (typeof interval !== 'number') watch(interval, startInterval)

  onUnmounted(() => clearInterval(timer))
}
