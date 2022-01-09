<template>
  <a
    v-long-press
    class="entry-link"
    :href="href"
    @click="entryClicked"
    @contextmenu="entryContextMenu"
    @long-press="entryContextMenu"
  >
    <slot />
  </a>
</template>
<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'

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

const router = useRouter()

const emit = defineEmits(['click', 'menu'])

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
