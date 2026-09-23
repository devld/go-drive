<template>
  <FilePreviewHost
    :entry="entry"
    :previewer="Model3dPreview"
    :load="load"
    :make-props="makeProps"
    @close="emit('close')"
  />
</template>
<script setup lang="ts">
import FilePreviewHost from '../FilePreviewHost.vue'
import Model3dPreview from '@go-drive/previewers/model3d'
import { fileUrl, getBlobContent, getEntry } from '@/api'
import { Entry } from '@/types'
import { dir, pathClean, pathJoin } from '@/utils'

const props = defineProps({
  entry: { type: Object as PropType<Entry>, required: true },
  entries: { type: Array as PropType<Entry[]>, default: () => [] },
})
const emit = defineEmits<{ (e: 'close'): void }>()

const load = (entry: Entry) =>
  Promise.resolve(fileUrl(entry.path, entry.meta, { useProxy: 'cors' }))

const resolveResource = async (uri: string) => {
  if (/^(?:https?:|data:|blob:)/i.test(uri)) {
    const response = await fetch(uri)
    if (!response.ok) throw new Error(`Unable to load model resource: ${uri}`)
    return response.arrayBuffer()
  }
  const resource = pathClean(pathJoin(dir(props.entry.path), uri.split(/[?#]/)[0]))
  const resourceEntry =
    props.entries.find((entry) => entry.path === resource) ??
    (await getEntry(resource))
  return getBlobContent(resourceEntry.path, resourceEntry.meta).then((blob) =>
    blob.arrayBuffer()
  )
}

const makeProps = (data: unknown) => ({
  src: data as string,
  filename: props.entry.name,
  resolveResource,
})
</script>
