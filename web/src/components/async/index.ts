import {
  AsyncComponentLoader,
  Component,
  ComponentPublicInstance,
  defineAsyncComponent,
  defineComponent,
  h,
} from 'vue'
import { useI18n } from '@go-drive/i18n'
import { LoadingState } from '@go-drive/utils'

const AsyncLoadingState = defineComponent(() => {
  const { t } = useI18n()

  return () => h(LoadingState, { variant: 'panel', text: t('app.loading') })
})

export function wrapAsyncComponent<
  T extends Component = {
    new (): ComponentPublicInstance
  }
>(loader: AsyncComponentLoader<T>) {
  return defineAsyncComponent({
    loader,
    loadingComponent: AsyncLoadingState,
  })
}
