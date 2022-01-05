<template>
  <router-link
    v-if="link"
    v-long-press
    class="entry-link"
    :to="link"
    @click="entryClicked"
    @contextmenu="entryContextMenu"
    @long-press="entryContextMenu"
  >
    <slot />
  </router-link>
  <a
    v-else
    v-long-press
    class="entry-link"
    href="javascript:;"
    @click="entryClicked"
    @contextmenu="entryContextMenu"
    @long-press="entryContextMenu"
  >
    <slot />
  </a>
</template>
<script setup>
import { computed } from 'vue'

const props = defineProps({
  entry: {
    type: Object,
  },
  path: {
    type: String,
  },
  getLink: {
    type: Function,
  },
})

const emit = defineEmits(['click', 'menu'])

const link = computed(() => {
  let link
  if (props.entry) {
    link = props.getLink?.(props.entry)
  } else if (typeof props.path === 'string') {
    link = props.getLink?.(props.path)
  }
  return link || ''
})

const entryClicked = (event) => {
  emit('click', {
    entry: props.entry,
    path: props.path,
    event,
  })
}
const entryContextMenu = (event) => {
  emit('menu', {
    entry: props.entry,
    path: props.path,
    event,
  })
}
</script>
