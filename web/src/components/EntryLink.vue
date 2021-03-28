<template>
  <a
    class="entry-link"
    :href="link"
    @click="entryClicked"
    v-long-press
    @contextmenu="entryContextMenu"
    @long-press="entryContextMenu"
  >
    <slot />
  </a>
</template>
<script>
import { makeEntryLink, getDirEntryLink } from '@/utils/routes'
import router from '@/router'

const ROUTER_MODE = router.mode
const routePathPrefix = ROUTER_MODE === 'hash' ? '#' : ''

export default {
  name: 'EntryLink',
  props: {
    entry: {
      type: Object,
    },
    path: {
      type: String,
    },
  },
  computed: {
    link() {
      if (this.entry) return routePathPrefix + makeEntryLink(this.entry)
      if (typeof this.path === 'string') {
        return routePathPrefix + getDirEntryLink(this.path)
      }
      return 'javascript:;'
    },
  },
  methods: {
    entryClicked(event) {
      this.$emit('click', {
        entry: this.entry,
        path: this.path,
        event,
      })
    },
    entryContextMenu(event) {
      this.$emit('menu', {
        entry: this.entry,
        path: this.path,
        event,
      })
    },
  },
}
</script>
