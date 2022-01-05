<template>
  <entry-explorer base-path="/_" />
</template>
<script>
export default { name: 'HomePage' }
</script>
<script setup>
import EntryExplorer from '@/views/EntryExplorer/index.vue'

defineProps({
  path: {
    type: String,
    required: true,
  },
})
</script>
