<template>
  <FilePreviewHost
    :entry="entry"
    :previewer="SpreadsheetPreview"
    :load="load"
    :make-props="makeProps"
    @close="emit('close')"
  />
</template>
<script setup lang="ts">
import FilePreviewHost from '../FilePreviewHost.vue'
import SpreadsheetPreview from '@go-drive/previewers/spreadsheet'
import { getContent } from '@/api'
import { Entry } from '@/types'

const props = defineProps({
  entry: { type: Object as PropType<Entry>, required: true },
})
const emit = defineEmits<{ (e: 'close'): void }>()

const load = (entry: Entry) => getContent(entry.path, entry.meta)
const makeProps = (data: unknown) => ({
  content: data as string,
  delimiter: props.entry.name.toLowerCase().endsWith('.tsv') ? '\t' : ',',
})
</script>
