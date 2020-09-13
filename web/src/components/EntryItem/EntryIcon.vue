<template>
  <i-icon @click="$emit('click', $event)" class="entry-icon" :svg="entryIcon" />
</template>
<script>
import { getIconSVG } from './file-icon'

export default {
  name: 'EntryIcon',
  props: {
    entry: {
      type: Object,
      required: true
    }
  },
  computed: {
    entryIcon () {
      return getIconSVG(this.entry)
    }
  }
}
</script>
