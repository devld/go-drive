<template>
  <div
    ref="element"
    class="preview-image-gallery"
    :class="{ loading: loading }"
  >
    <LoadingState
      v-if="loading"
      variant="overlay"
      text="Loading"
      :surface="false"
    />
  </div>
</template>
<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import PhotoSwipe from 'photoswipe'
import 'photoswipe/dist/photoswipe.css'
import { LoadingState } from '@go-drive/utils'

export interface GalleryImage {
  src: string
  width?: number
  height?: number
}

const props = withDefaults(
  defineProps<{
    images: GalleryImage[]
    initialIndex?: number
  }>(),
  { initialIndex: 0 }
)

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'change', index: number): void
}>()

const element = ref<HTMLDivElement | null>(null)
const loading = ref(false)
let gallery: PhotoSwipe | undefined

onMounted(() => {
  if (!element.value || !props.images.length) return
  gallery = new PhotoSwipe({
    appendToEl: element.value,
    dataSource: props.images,
    index: Math.max(0, Math.min(props.initialIndex, props.images.length - 1)),
    loop: true,
    wheelToZoom: true,
    pinchToClose: true,
  })
  gallery.on('gettingData', ({ data, index }) => {
    if (!gallery || (data.width && data.height)) return
    loading.value = true
    const image = new Image()
    image.onload = () => {
      data.width = image.width
      data.height = image.height
      loading.value = false
      gallery?.refreshSlideContent(index)
    }
    image.onerror = () => {
      loading.value = false
    }
    image.src = data.src ?? ''
  })
  gallery.on('close', () => emit('close'))
  gallery.on('change', () => emit('change', gallery!.currIndex))
  gallery.init()
})

onUnmounted(() => {
  gallery?.destroy()
  gallery = undefined
})
</script>
<style lang="scss">
.preview-image-gallery {
  position: relative;
  width: 100%;
  height: 100%;
  background: #000;
}
</style>
