<template>
  <span class="entry-icon">
    <svg class="icon" aria-hidden="true">
      <use :xlink:href="entryIcon" />
    </svg>
  </span>
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
