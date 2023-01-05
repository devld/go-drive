<template>
  <div class="pdf-preview-view">
    <HandlerTitleBar :title="filename" @close="emit('close')" />

    <iframe
      ref="iframe"
      :key="previewURL"
      class="pdf-preview-iframe"
      :src="previewURL"
      frameborder="0"
    ></iframe>
  </div>
</template>
<script setup lang="ts">
import { fileUrl } from '@/api'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { Entry } from '@/types'
import { buildURL, filename as filenameFn } from '@/utils'
import { computed } from 'vue'

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  entries: { type: Array as PropType<Entry[]> },
})

const emit = defineEmits<{ (e: 'close'): void }>()

const path = computed(() => props.entry.path)
const filename = computed(() => filenameFn(path.value))
const fileURL = computed(() =>
  fileUrl(path.value, props.entry.meta, { useProxy: 'cors' })
)

const previewURL = computed(() =>
  buildURL('pdf.js/web/viewer.html', { file: fileURL.value })
)
</script>
<style lang="scss">
.pdf-preview-view {
  position: relative;
  overflow: hidden;
  width: 100vw;
  height: 100%;
  padding-top: 48px;
  background-color: var(--secondary-bg-color);
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
  box-sizing: border-box;

  .handler-title-bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }

  .pdf-preview-iframe {
    width: 100%;
    height: 100%;
  }
}
</style>
