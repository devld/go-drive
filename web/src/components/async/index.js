import { defineAsyncComponent } from 'vue'
import LoadingComponent from './LoadingComponent.vue'

export function wrapAsyncComponent(loader) {
  return defineAsyncComponent({
    loader,
    loadingComponent: LoadingComponent,
  })
}
