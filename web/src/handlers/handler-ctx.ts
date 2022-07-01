import { useAppStore } from '@/store'
import { computed } from 'vue'

export const useHandlerCtx = () => {
  const store = useAppStore()

  return computed(() => ({
    user: store.user,
    config: store.config!,
    options: store.config!.options,
  }))
}
