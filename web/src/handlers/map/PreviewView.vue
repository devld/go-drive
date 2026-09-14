<template>
  <FilePreviewHost
    :entry="entry"
    :previewer="MapPreview"
    :load="load"
    :make-props="makeProps"
    @close="emit('close')"
  />
</template>
<script setup lang="ts">
import FilePreviewHost from '../FilePreviewHost.vue'
import MapPreview from '@go-drive/previewers/map'
import { getContent } from '@/api'
import { Entry } from '@/types'
import { filenameExt } from '@/utils'

const props = defineProps({
  entry: { type: Object as PropType<Entry>, required: true },
})
const emit = defineEmits<{ (e: 'close'): void }>()

const load = (entry: Entry) => getContent(entry.path, entry.meta)
const makeProps = (data: unknown) => ({
  data: data as string,
  format: filenameExt(props.entry.name) as 'geojson' | 'gpx' | 'kml',
})
</script>
