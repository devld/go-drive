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
import {
  ARTIFACT_THUMBNAIL,
  artifactRefUrl,
  prepareArtifact,
} from '@/api/artifact'
import { entryMatches, taskDone } from '@/utils'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
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

const thumbnailConfig = computed(() => store.config?.artifact?.thumbnail)
const entryIcon = computed(() => getEntryIcon(props.entry))
const supportThumbnail = computed(() => {
  const entry = props.entry
  const extensions = thumbnailConfig.value?.extensions
  if (!extensions) return false
  return entry.type === 'dir'
    ? extensions.includes('/')
    : entryMatches(entry, extensions)
})
const thumbnail = ref<string>()
let thumbnailRequest = 0

watch(
  () =>
    [
      props.showThumbnail,
      props.entry.path,
      props.entry.meta?.hasThumbnail,
      supportThumbnail.value,
    ] as const,
  async ([show, path, hasThumbnail, supported]) => {
    const request = ++thumbnailRequest
    thumbnail.value = undefined
    err.value = null
    thumbnailLoaded.value = false
    if (!show || (!supported && !hasThumbnail)) return
    try {
      const prepared = await prepareArtifact(
        path,
        props.entry.meta,
        ARTIFACT_THUMBNAIL
      )
      const info =
        'info' in prepared ? prepared.info : await taskDone(prepared.task)
      if (request !== thumbnailRequest || !info || !info.ref) return
      thumbnail.value = artifactRefUrl(
        path,
        props.entry.meta,
        ARTIFACT_THUMBNAIL,
        info.ref
      )
    } catch {
      if (request === thumbnailRequest) err.value = new Event('error')
    }
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  thumbnailRequest++
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
