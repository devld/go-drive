<template>
  <entry-explorer :base-path="HOME_ROUTE_PREFIX" />
</template>
<script>
export default { name: 'HomePage' }
</script>
<script setup>
import { HOME_ROUTE_PREFIX } from '@/config'
import EntryExplorer from '@/views/EntryExplorer/index.vue'

defineProps({
  path: {
    type: String,
    required: true,
  },
})
</script>
