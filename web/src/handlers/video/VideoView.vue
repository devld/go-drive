<template>
  <div class="video-view-page">
    <HandlerTitleBar :title="entry.name" @close="emit('close')" />
    <video
      :src="fileUrl(entry.path, entry.meta, { useProxy: 'referrer' })"
      controls
    />
  </div>
</template>
<script setup lang="ts">
import { fileUrl } from '@/api'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import { Entry } from '@/types'

defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  entries: { type: Array as PropType<Entry[]> },
})

const emit = defineEmits<{ (e: 'close'): void }>()
</script>
<style lang="scss">
.video-view-page {
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
