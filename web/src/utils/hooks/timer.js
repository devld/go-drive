import { onUnmounted, onMounted } from 'vue'

/**
 *
 * @param {Function} callback
 * @param {number} interval
 * @param {boolean} immediate
 */
export const useInterval = (callback, interval, immediate) => {
  let t

  onMounted(() => {
    if (immediate) {
      callback()
    }

    t = setInterval(() => {
      callback()
    }, interval)
  })

  onUnmounted(() => {
    clearInterval(t)
  })
}
