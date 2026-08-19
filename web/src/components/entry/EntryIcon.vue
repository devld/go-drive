<template>
  <span
    class="entry-icon"
    :class="`entry-icon-${entry.type}`"
    @click="emit('click', $event)"
  >
    <Icon
      v-show="entry.type === 'dir' || !thumbnailLoaded"
      :name="icon || entryIcon"
    />
    <img
      v-if="showThumbnail && thumbnail && !err"
      v-lazy-src="thumbnail"
      class="entry-icon__thumbnail"
      :class="{ 'entry-icon__thumbnail--loaded': thumbnailLoaded }"
      :alt="entry.name"
      @load="onLoad"
      @error="onError"
    />
  </span>
</template>
<script setup lang="ts">
import { getEntryIcon } from './file-icon'
import { fileThumbnail } from '@/api'
import { filenameExt } from '@/utils'
import { ref, computed, watch } from 'vue'
import { Entry } from '@/types'
import { useAppStore } from '@/store'
import type { IconName } from '@/components/icons'

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
    required: true,
  },
  icon: {
    type: String as PropType<IconName>,
  },
  showThumbnail: {
    type: Boolean,
    default: true,
  },
})

const emit = defineEmits<{ (e: 'click', event: MouseEvent): void }>()

const err = ref<Event | null>(null)
const thumbnailLoaded = ref(false)

const store = useAppStore()

const thumbnailConfig = computed(() => store.config!.thumbnail)
const entryIcon = computed(() => getEntryIcon(props.entry))
const supportThumbnail = computed(() => {
  const entry = props.entry
  const ext = entry.type === 'dir' ? '/' : filenameExt(entry.name)
  return !!thumbnailConfig.value.extensions?.[ext]
})
const thumbnail = computed(() => {
  const url = props.entry.meta.thumbnailUrl
  if (url) return url
  if (supportThumbnail.value || props.entry.meta.selfThumbnail) {
    return fileThumbnail(props.entry.path, props.entry.meta)
  }
  return undefined
})

watch(thumbnail, () => {
  err.value = null
  thumbnailLoaded.value = false
})

const onLoad = () => (thumbnailLoaded.value = true)
const onError = (e: Event) => {
  thumbnailLoaded.value = false
  err.value = e
}
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
  visibility: hidden;
}

.entry-icon__thumbnail--loaded {
  visibility: visible;
}

.entry-icon-dir {
  .entry-icon__thumbnail {
    top: 60%;
    left: 60%;
    width: 50%;
    height: 50%;
    transform: translate(-50%, -50%);
  }
}
</style>
