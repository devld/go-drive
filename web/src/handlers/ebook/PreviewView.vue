<template>
  <FilePreviewHost
    :entry="entry"
    :previewer="EpubPreview"
    :load="load"
    :make-props="makeProps"
    @close="emit('close')"
  />
</template>
<script setup lang="ts">
import FilePreviewHost from '../FilePreviewHost.vue'
import EpubPreview from '@go-drive/previewers/ebook'
import { fileUrl } from '@/api'
import { Entry } from '@/types'

const emit = defineEmits<{ (e: 'close'): void }>()
defineProps({
  entry: { type: Object as PropType<Entry>, required: true },
})
const load = (entry: Entry) =>
  Promise.resolve(fileUrl(entry.path, entry.meta, { useProxy: 'cors' }))
const makeProps = (data: unknown) => ({ src: data as string })
</script>
