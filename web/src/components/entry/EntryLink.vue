<template>
  <a
    v-long-press
    class="entry-link"
    :href="href"
    :draggable="draggable ? 'true' : undefined"
    @dragstart="onDragStart"
    @dragover="onDragOver"
    @drop="onDrop"
    @click="entryClicked"
    @contextmenu="entryContextMenu"
    @long-press="entryContextMenu"
  >
    <slot />
  </a>
</template>
<script setup lang="ts">
import { Entry } from '@/types'
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { EntryEventData, GetLinkFn } from '.'

const props = defineProps({
  entry: {
    type: Object as PropType<Entry>,
  },
  path: {
    type: String,
  },
  getLink: {
    type: Function as PropType<GetLinkFn>,
  },
  draggable: {
    type: Boolean,
  },
})

const router = useRouter()

const emit = defineEmits<{
  (e: 'click', data: EntryEventData): void
  (e: 'menu', data: EntryEventData): void
  (e: 'dragstart', data: EntryEventData): void
  (e: 'dragover', data: EntryEventData): void
  (e: 'drop', data: EntryEventData): void
}>()

const link = computed(() => {
  let link
  if (props.entry) {
    link = props.getLink?.(props.entry)
  } else if (typeof props.path === 'string') {
    link = props.getLink?.(props.path)
  }
  return link
})

const href = computed(() => {
  const routeLink = link.value
  if (!routeLink) return 'javascript:;'
  const route = router.resolve(routeLink)
  return route.href
})

const entryClicked = (event: MouseEvent) => {
  emit('click', {
    entry: props.entry,
    path: props.path,
    event,
  })
}

const entryContextMenu = (event: MouseEvent) => {
  emit('menu', {
    entry: props.entry,
    path: props.path,
    event,
  })
}

const onDragStart = (event: DragEvent) => {
  emit('dragstart', {
    entry: props.entry,
    path: props.path,
    event,
  })
}

const onDragOver = (event: DragEvent) => {
  emit('dragover', {
    entry: props.entry,
    path: props.path,
    event,
  })
}

const onDrop = (event: DragEvent) => {
  emit('drop', {
    entry: props.entry,
    path: props.path,
    event,
  })
}
</script>
