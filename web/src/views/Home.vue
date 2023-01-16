<template>
  <EntryExplorer :base-path="EXPLORER_PATH_BASE" />
</template>
<script lang="ts">
import { EXPLORER_PATH_BASE } from '@/config'

export default { name: 'HomePage' }
</script>
<script setup lang="ts">
import EntryExplorer from '@/views/EntryExplorer/index.vue'

defineProps({
  path: {
    type: String,
    required: true,
  },
})
</script>
