<template>
  <span class="entry-icon" @click="$emit('click', $event)">
    <i-icon :svg="icon || entryIcon" />
    <img
      class="entry-icon__thumbnail"
      v-if="showThumbnail && thumbnail && !err"
      v-lazy="thumbnail"
      :alt="entry.name"
      @error="onError"
    />
  </span>
</template>
<script>
import { supportThumbnail } from '@/utils'
import { getIconSVG } from './file-icon'
import { fileThumbnail } from '@/api'

export default {
  name: 'EntryIcon',
  props: {
    entry: {
      type: Object,
      required: true
    },
    icon: {
      type: String
    },
    showThumbnail: {
      type: Boolean,
      default: true
    }
  },
  data () {
    return {
      err: null
    }
  },
  computed: {
    entryIcon () {
      return getIconSVG(this.entry)
    },
    thumbnail () {
      return this.entry.meta.thumbnail ||
        (supportThumbnail(this.entry) && fileThumbnail(this.entry.path, this.entry.meta.access_key))
    }
  },
  methods: {
    onError (e) {
      this.err = e
    }
  }
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
}
</style>
