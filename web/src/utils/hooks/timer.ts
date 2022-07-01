import { onUnmounted, onMounted } from 'vue'

export const useInterval = (
  callback: Fn<void>,
  interval: number,
  immediate: boolean
) => {
  let t: number

  onMounted(() => {
    if (immediate) {
      callback()
    }

    t = setInterval(() => {
      callback()
    }, interval) as unknown as number
  })

  onUnmounted(() => {
    clearInterval(t)
  })
}
