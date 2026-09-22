<template>
  <span
    class="entry-icon"
    :class="`entry-icon-${entry.type}`"
    @click="emit('click', $event)"
  >
    <Icon
      v-show="entry.type === 'dir' || !showThumbnail || !thumbnailLoaded"
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
import { artifactURL } from '@/api'
import { entryMatches } from '@/utils'
import { computed, ref, watch } from 'vue'
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

const entryIcon = computed(() => getEntryIcon(props.entry))
const thumbnailConfig = computed(() =>
  props.showThumbnail ? store.config?.artifact?.thumbnail : undefined
)
const supportThumbnail = computed(() => {
  if (!props.showThumbnail) return false
  const entry = props.entry
  const extensions = thumbnailConfig.value?.extensions
  if (!extensions) return false
  return entry.type === 'dir'
    ? extensions.includes('/')
    : entryMatches(entry, extensions)
})
const thumbnail = computed(() => {
  if (!props.showThumbnail) return undefined
  if (supportThumbnail.value || props.entry.meta.hasThumbnail) {
    return artifactURL(props.entry.path, props.entry.meta, 'thumbnail')
  }
  return undefined
})

watch(thumbnail, () => {
  if (!props.showThumbnail) return
  err.value = null
  thumbnailLoaded.value = false
})

watch(
  () => props.showThumbnail,
  () => {
    thumbnailLoaded.value = false
  }
)

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
