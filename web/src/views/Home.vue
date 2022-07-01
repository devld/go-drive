<template>
  <EntryExplorer base-path="/_" />
</template>
<script lang="ts">
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
