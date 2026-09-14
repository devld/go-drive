<template>
  <ImageGallery
    :images="images"
    :initial-index="currentIndex"
    @close="emit('close')"
    @change="changeImage"
  />
</template>
<script setup lang="ts">
import ImageGallery from '@go-drive/previewers/image'
import { fileUrl } from '@/api'
import { Entry } from '@/types'
import { dir, filename as filenameFn, filenameExt, pathJoin } from '@/utils'
import { computed } from 'vue'
import { EntryHandlerContext } from '../types'

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

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'entry-change', path: string): void
}>()

const images = computed(() => {
  const basePath = dir(props.entry.path)
  return props.entries
    .filter(
      (entry) =>
        entry.type === 'file' &&
        props.ctx.options['web.imageFileExts'].includes(filenameExt(entry.name))
    )
    .map((entry) => ({
      src: fileUrl(pathJoin(basePath, entry.name), entry.meta, {
        useProxy: 'referrer',
      }),
    }))
})

const currentIndex = computed(() => {
  const name = filenameFn(props.entry.path)
  const entries = props.entries.filter(
    (entry) =>
      entry.type === 'file' &&
      props.ctx.options['web.imageFileExts'].includes(filenameExt(entry.name))
  )
  return entries.findIndex((entry) => entry.name === name)
})

const changeImage = (index: number) => {
  const entry = props.entries
    .filter(
      (item) =>
        item.type === 'file' &&
        props.ctx.options['web.imageFileExts'].includes(filenameExt(item.name))
    )
    .at(index)
  if (entry) emit('entry-change', entry.path)
}
</script>
