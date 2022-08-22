<template>
  <div
    ref="psEl"
    class="image-view-page"
    :class="{ loading: isCurrentImageSizeLoading }"
  ></div>
</template>
<script setup lang="ts">
import { fileUrl } from '@/api'
import PhotoSwipe from 'photoswipe'
import { filenameExt, filename as filenameFn, dir, pathJoin } from '@/utils'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Entry } from '@/types'
import { EntryHandlerContext } from '../types'
import { DEFAULT_IMAGE_FILE_EXTS } from '@/config'

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  entries: {
    type: Array as PropType<Entry[]>,
    required: true,
  },
  ctx: {
    type: Object as PropType<EntryHandlerContext>,
    required: true,
  },
})

function isSupportedImageExt(ext: string) {
  return (
    props.ctx.options['web.imageFileExts'] || DEFAULT_IMAGE_FILE_EXTS
  ).includes(ext)
}

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'entry-change', v: string): void
}>()

const images = computed(() =>
  props.entries.filter(
    (e) => e.type === 'file' && isSupportedImageExt(filenameExt(e.name))
  )
)

const path = computed(() => props.entry.path)

const filename = computed(() => filenameFn(path.value))

const psEl = ref<HTMLElement | null>(null)

let ps: PhotoSwipe | undefined

const index = ref(0)

const imageSizeLoading = ref(new Set<number>())
const isCurrentImageSizeLoading = computed(() =>
  imageSizeLoading.value.has(index.value)
)

const initPhotoSwipe = () => {
  index.value = images.value.findIndex((f) => f.name === filename.value)

  const basePath = dir(path.value)
  ps = new PhotoSwipe({
    appendToEl: psEl.value!,
    dataSource: images.value.map((i) => ({
      src: fileUrl(pathJoin(basePath, i.name), i.meta, {
        useProxy: 'referrer',
      }),
    })),
    index: index.value,
    loop: true,
    wheelToZoom: true,
    pinchToClose: true,
  })
  ps.on('gettingData', ({ data, index }) => {
    if (!ps) return
    // https://github.com/dimsemenov/PhotoSwipe/issues/796
    if (data.width! > 0 && data.height! > 0) return
    imageSizeLoading.value.add(index)
    const img = new Image()
    img.onload = function () {
      data.width = img.width
      data.height = img.height
      ps?.refreshSlideContent(index)
      imageSizeLoading.value.delete(index)
    }
    img.src = data.src!
  })
  ps.on('close', () => {
    if (!ps) return
    emit('close')
    ps = undefined
  })
  ps.on('change', () => {
    index.value = ps!.currIndex
    emit('entry-change', images.value[index.value].path)
  })
  ps.init()
}

onMounted(() => {
  initPhotoSwipe()
})

onUnmounted(() => {
  if (!ps) return
  const ps_ = ps
  ps = undefined
  ps_.destroy()
})
</script>
<style lang="scss">
@import url('photoswipe/dist/photoswipe.css');
.image-view-page {
  width: 100vw;
  height: 100vh;

  &.loading {
    .pswp__img {
      visibility: hidden;
    }
  }
}
</style>
