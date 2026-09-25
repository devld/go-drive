<template>
  <FilePreviewHost
    :entry="entry"
    :previewer="WebPreview"
    :load="load"
    :make-props="makeProps"
    @close="emit('close')"
  />
</template>
<script setup lang="ts">
import FilePreviewHost from '../FilePreviewHost.vue'
import WebPreview from './WebPreview.vue'
import { getContent } from '@/api'
import { Entry } from '@/types'
import { filenameExt } from '@/utils'

const props = defineProps({
  entry: { type: Object as PropType<Entry>, required: true },
})
const emit = defineEmits<{ (e: 'close'): void }>()

const load = (entry: Entry) => getContent(entry.path, entry.meta)
const makeProps = (data: unknown) => ({
  content: data as string,
  type: filenameExt(props.entry.name) === 'svg' ? 'svg' : 'html',
})
</script>
