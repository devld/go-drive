<template>
  <div class="iframe-preview-view">
    <HandlerTitleBar :title="filename" @close="emit('close')">
      <template #actions>
        <select v-if="services.length > 1" v-model="serviceIndex">
          <option v-for="(s, i) in services" :key="s.name" :value="i">
            {{ s.name }}
          </option>
        </select>
      </template>
    </HandlerTitleBar>

    <iframe
      v-if="previewURL"
      ref="iframe"
      :key="previewURL"
      class="preview-iframe"
      :src="previewURL"
      frameborder="0"
    ></iframe>
  </div>
</template>
<script setup lang="ts">
import { fileUrl } from '@/api'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { Entry } from '@/types'
import { entryMatches, filename as filenameFn, filenameExt } from '@/utils'
import { computed, ref, watch } from 'vue'
import { EntryHandlerContext } from '../types'

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  entries: { type: Array as PropType<Entry[]> },
  ctx: {
    type: Object as PropType<EntryHandlerContext>,
    required: true,
  },
})

const emit = defineEmits<{ (e: 'close'): void }>()

const path = computed(() => props.entry.path)
const filename = computed(() => filenameFn(path.value))
const fileExt = computed(() => filenameExt(filename.value))
const fileURL = computed(() => fileUrl(path.value, props.entry.meta))

const services = computed(() =>
  props.ctx.options['web.externalFileViewers'].filter((e) =>
    entryMatches(props.entry, e.exts)
  )
)

const storageKey = computed(() => `iframe-preview:${fileExt.value}`)
const serviceIndex = ref(+(localStorage.getItem(storageKey.value) ?? '0') || 0)
if (!services.value[serviceIndex.value]) serviceIndex.value = 0
watch(
  () => serviceIndex.value,
  (v) => {
    localStorage.setItem(storageKey.value, `${v}`)
  }
)

const service = computed(() => services.value[serviceIndex.value])

const previewURL = computed(() => {
  if (!service.value) return
  let url = service.value.url

  url = url.replace('{URL}', encodeURIComponent(fileURL.value))
  url = url.replace('{NAME}', encodeURIComponent(filename.value))
  return url
})
</script>
<style lang="scss">
.iframe-preview-view {
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

  .preview-iframe {
    width: 100%;
    height: 100%;
  }
}
</style>
