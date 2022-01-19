<template>
  <entry-explorer :base-path="basePath" :ctx="ctx">
    <template #pathBarRoot>{{ $t('p.share.root_path') }}</template>
  </entry-explorer>
</template>
<script>
export default { name: 'SharedFolder' }
</script>
<script setup>
import { SHARED_FOLDER_ROUTE_PREFIX } from '@/config'
import EntryExplorer from '@/views/EntryExplorer/index.vue'
import { computed } from 'vue'

const props = defineProps({
  sharedId: {
    type: String,
    required: true,
  },
  path: {
    type: String,
    required: true,
  },
})

const basePath = computed(
  () => `${SHARED_FOLDER_ROUTE_PREFIX}/${props.sharedId}`
)

const ctx = computed(() => ({
  sharedId: props.sharedId,
}))
</script>
