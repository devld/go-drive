import {
  AsyncComponentLoader,
  Component,
  ComponentPublicInstance,
  defineAsyncComponent,
} from 'vue'
import LoadingComponent from './LoadingComponent.vue'

export function wrapAsyncComponent<
  T extends Component = {
    new (): ComponentPublicInstance
  }
>(loader: AsyncComponentLoader<T>) {
  return defineAsyncComponent({
    loader,
    loadingComponent: LoadingComponent,
  })
}
