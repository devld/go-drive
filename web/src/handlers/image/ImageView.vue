<template>
  <div class="image-view-page">
    <!-- Root element of PhotoSwipe. Must have class pswp. -->
    <div ref="psEl" class="pswp" tabindex="-1" role="dialog" aria-hidden="true">
      <!-- Background of PhotoSwipe.
      It's a separate element as animating opacity is faster than rgba().-->
      <div class="pswp__bg"></div>

      <!-- Slides wrapper with overflow:hidden. -->
      <div class="pswp__scroll-wrap">
        <!-- Container that holds slides.
            PhotoSwipe keeps only 3 of them in the DOM to save memory.
        Don't modify these 3 pswp__item elements, data is added later on.-->
        <div class="pswp__container">
          <div class="pswp__item"></div>
          <div class="pswp__item"></div>
          <div class="pswp__item"></div>
        </div>

        <!-- Default (PhotoSwipeUI_Default) interface on top of sliding area. Can be changed. -->
        <div class="pswp__ui pswp__ui--hidden">
          <div class="pswp__top-bar">
            <!--  Controls are self-explanatory. Order can be changed. -->

            <div class="pswp__counter"></div>
            <button
              class="pswp__button pswp__button--close"
              title="Close (Esc)"
            ></button>
            <button
              class="pswp__button pswp__button--fs"
              title="Toggle fullscreen"
            ></button>
            <button
              class="pswp__button pswp__button--zoom"
              title="Zoom in/out"
            ></button>

            <!-- Preloader demo https://codepen.io/dimsemenov/pen/yyBWoR -->
            <!-- element will get class pswp__preloader--active when preloader is running -->
            <div class="pswp__preloader">
              <div class="pswp__preloader__icn">
                <div class="pswp__preloader__cut">
                  <div class="pswp__preloader__donut"></div>
                </div>
              </div>
            </div>
          </div>

          <div
            class="pswp__share-modal pswp__share-modal--hidden pswp__single-tap"
          >
            <div class="pswp__share-tooltip"></div>
          </div>

          <button
            class="pswp__button pswp__button--arrow--left"
            title="Previous (arrow left)"
          ></button>

          <button
            class="pswp__button pswp__button--arrow--right"
            title="Next (arrow right)"
          ></button>

          <div class="pswp__caption">
            <div class="pswp__caption__center"></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { fileUrl } from '@/api'
import PhotoSwipe from 'photoswipe'
import PhotoSwipeUIDefault from 'photoswipe/dist/photoswipe-ui-default'
import { filenameExt, filename as filenameFn, dir, pathJoin } from '@/utils'
import { computed, onMounted, ref } from 'vue'
import { Entry } from '@/types'

function isSupportedImageExt(ext: string) {
  return ['jpg', 'jpeg', 'png', 'gif'].includes(ext)
}

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  entries: {
    type: Array as PropType<Entry[]>,
    required: true,
  },
})

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

let index: number

const initPhotoSwipe = () => {
  index = images.value.findIndex((f) => f.name === filename.value)
  const basePath = dir(path.value)
  const ps = new PhotoSwipe(
    psEl.value!,
    PhotoSwipeUIDefault,
    images.value.map((i) => ({
      src: fileUrl(pathJoin(basePath, i.name), i.meta, {
        useProxy: 'referrer',
      }),
      w: 0,
      h: 0,
    })),
    {
      history: false,
      index,
      loop: false,
    }
  )
  ps.listen('gettingData', (index: number, item: any) => {
    // https://github.com/dimsemenov/PhotoSwipe/issues/796
    if (item.w > 0 && item.h > 0) return
    const img = new Image()
    img.onload = function () {
      item.w = img.width
      item.h = img.height
      ps.updateSize(true)
    }
    img.src = item.src
  })
  ps.listen('close', () => {
    emit('close')
  })
  ps.listen('beforeChange', (offset: number) => {
    if (!offset) return
    let newIndex = (index += offset)
    if (newIndex < 0) newIndex += images.value.length
    if (newIndex >= images.value.length) newIndex -= images.value.length
    index = newIndex
    emit('entry-change', images.value[index].path)
  })
  ps.init()
}

onMounted(() => {
  initPhotoSwipe()
})
</script>
<style lang="scss">
@import url('photoswipe/dist/photoswipe.css');
@import url('photoswipe/dist/default-skin/default-skin.css');
.image-view-page {
  width: 100vw;
  height: 100vh;
}
</style>
