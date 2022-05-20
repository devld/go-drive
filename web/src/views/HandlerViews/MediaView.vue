<template>
  <div class="media-view-page">
    <handler-title-bar :title="entry.name" @close="emit('close')" />
    <video
      :src="fileUrl(entry.path, entry.meta, { useProxy: 'referrer' })"
      controls
    />
  </div>
</template>
<script setup>
import { fileUrl } from '@/api'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'

defineProps({
  entry: {
    type: Object,
    required: true,
  },
  entries: { type: Array },
})

const emit = defineEmits(['close'])
</script>
<style lang="scss">
.media-view-page {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding-top: 48px;
  background-color: var(--secondary-bg-color);

  .handler-title-bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }
  video {
    max-width: 90vw;
    max-height: 70vh;
    outline: none;
  }
}
</style>
