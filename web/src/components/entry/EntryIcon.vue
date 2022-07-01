<template>
  <span class="entry-icon" @click="emit('click', $event)">
    <Icon :svg="icon || entryIcon" />
    <img
      v-if="showThumbnail && thumbnail && !err"
      v-lazy-src="thumbnail"
      class="entry-icon__thumbnail"
      :alt="entry.name"
      @error="onError"
    />
  </span>
</template>
<script setup lang="ts">
import { getIconSVG } from './file-icon'
import { fileThumbnail } from '@/api'
import { filenameExt } from '@/utils'
import { ref, computed } from 'vue'
import { Entry } from '@/types'
import { useAppStore } from '@/store'

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  icon: {
    type: String,
  },
  showThumbnail: {
    type: Boolean,
    default: true,
  },
})

const emit = defineEmits<{ (e: 'click', event: MouseEvent): void }>()

const err = ref<Event | null>(null)

const store = useAppStore()

const thumbnailConfig = computed(() => store.config!.thumbnail)
const entryIcon = computed(() => getIconSVG(props.entry))
const supportThumbnail = computed(() => {
  const entry = props.entry
  const ext = entry.type === 'dir' ? '/' : filenameExt(entry.name)
  return !!thumbnailConfig.value.extensions?.[ext]
})
const thumbnail = computed(() => {
  const t = props.entry.meta.thumbnail
  if (typeof t === 'string') return t
  if (supportThumbnail.value || t === true) {
    return fileThumbnail(props.entry.path, props.entry.meta)
  }
  return undefined
})

const onError = (e: Event) => (err.value = e)
</script>
<style lang="scss">
.entry-icon {
  position: relative;
  overflow: hidden;
  border-radius: 10px;
  display: inline-block;
  width: 42px;
  height: 42px;

  .icon {
    display: block;
    width: 100%;
    height: 100%;
  }
}

.entry-icon__thumbnail {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>
